package worker

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fiatjaf/narr/src/parser"
	"github.com/jmoiron/sqlx"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip05"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/nbd-wtf/go-nostr/nip23"
	"github.com/nbd-wtf/go-nostr/sdk"
	"github.com/nbd-wtf/go-nostr/sdk/hints/sqlite"
	"github.com/puzpuzpuz/xsync/v3"
)

var nostrSdk *sdk.System

func InitializeNostr(db *sql.DB) error {
	hdb, err := sqlite.NewSQLiteHints(sqlx.NewDb(db, "sqlite"))
	if err != nil {
		return err
	}

	nostrSdk = sdk.NewSystem(
		sdk.WithHintsDB(hdb),
	)

	return nil
}

func isItNostr(url string) (bool, *sdk.ProfileMetadata) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// check for nostr url prefixes
	if strings.HasPrefix(url, "nostr://") {
		url = url[8:]
	} else if strings.HasPrefix(url, "nostr:") {
		url = url[6:]
	} else {
		// only accept nostr: or nostr:// urls for now
		return false, nil
	}

	// check for npub or nprofile
	if prefix, _, err := nip19.Decode(url); err == nil {
		if prefix == "nprofile" || prefix == "npub" {
			profile, err := nostrSdk.FetchProfileFromInput(ctx, url)
			if err != nil {
				return false, nil
			}
			return true, &profile
		}
	}

	if nip05.IsValidIdentifier(url) {
		profile, err := nostrSdk.FetchProfileFromInput(ctx, url)
		if err != nil {
			return false, nil
		}
		return true, &profile
	}

	return false, nil
}

func discoverNostr(candidateUrl string) (bool, *DiscoverResult) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	yes, profile := isItNostr(candidateUrl)
	if yes {
		nprofile := profile.Nprofile(ctx, nostrSdk, 3)

		// get some feed items
		items, err := nostrListItems(profile)
		if err != nil {
			items = []parser.Item{}
		}

		return true, &DiscoverResult{
			FeedLink: fmt.Sprintf("nostr:%s", nprofile),
			Feed: &parser.Feed{
				Title:   profile.Name,
				SiteURL: fmt.Sprintf("https://njump.me/%s", nprofile),
				Items:   items,
			},
			Sources: []FeedSource{},
		}
	}
	return false, nil
}

func nostrListItems(profile *sdk.ProfileMetadata) ([]parser.Item, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	relays := nostrSdk.FetchOutboxRelays(ctx, profile.PubKey, 3)
	evchan := nostrSdk.Pool.SubManyEose(ctx, relays, nostr.Filters{
		{
			Authors: []string{profile.PubKey},
			Kinds:   []int{nostr.KindArticle},
			Limit:   32,
		},
	})
	feedItems := []parser.Item{}
	for event := range evchan {
		publishedAt := event.CreatedAt.Time()
		if paTag := event.Tags.GetFirst([]string{"published_at", ""}); paTag != nil && len(*paTag) >= 2 {
			i, err := strconv.ParseInt((*paTag)[1], 10, 64)
			if err != nil {
				publishedAt = time.Unix(i, 0)
			}
		}

		naddr, err := nip19.EncodeEntity(event.PubKey, event.Kind, event.Tags.GetD(), relays)
		if err != nil {
			continue
		}

		title := ""
		titleTag := event.Tags.GetFirst([]string{"title", ""})
		if titleTag != nil && len(*titleTag) >= 2 {
			title = (*titleTag)[1]
		} else {
			continue
		}

		image := ""
		imageTag := event.Tags.GetFirst([]string{"image", ""})
		if imageTag != nil && len(*imageTag) >= 2 {
			image = (*imageTag)[1]
		}

		// format content from markdown to html
		htmlContent := replaceNostrURLsWithHTMLTags(nip23.MarkdownToHTML(event.Content))

		feedItems = append(feedItems, parser.Item{
			GUID:     fmt.Sprintf("nostr:%s:%s", event.PubKey, event.Tags.GetD()),
			Date:     publishedAt,
			URL:      fmt.Sprintf("https://njump.me/%s", naddr),
			Content:  htmlContent,
			Title:    title,
			ImageURL: image,
		})

	}

	return feedItems, nil
}

var nostrEveryMatcher = regexp.MustCompile(`nostr:((npub|note|nevent|nprofile|naddr)1[a-z0-9]+)\b`)

func replaceNostrURLsWithHTMLTags(input string) string {
	// match and replace npup1, nprofile1, note1, nevent1, etc
	names := xsync.NewMapOf[string, string]()
	wg := sync.WaitGroup{}

	// first we run it without waiting for the results of getting the name as they will be async
	for _, match := range nostrEveryMatcher.FindAllString(input, len(input)+1) {
		nip19 := match[len("nostr:"):]

		if strings.HasPrefix(nip19, "npub1") || strings.HasPrefix(nip19, "nprofile1") {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
			defer cancel()
			wg.Add(1)
			go func() {
				metadata, _ := nostrSdk.FetchProfileFromInput(ctx, nip19)
				if metadata.Name != "" {
					names.Store(nip19, metadata.Name)
				}
				wg.Done()
			}()
		}
	}

	// in the second time now that we got all the names we actually perform replacement
	wg.Wait()
	return nostrEveryMatcher.ReplaceAllStringFunc(input, func(match string) string {
		nip19 := match[len("nostr:"):]
		firstChars := nip19[:8]
		lastChars := nip19[len(nip19)-4:]

		if strings.HasPrefix(nip19, "npub1") || strings.HasPrefix(nip19, "nprofile1") {
			name, _ := names.Load(nip19)
			if name != "" {
				return fmt.Sprintf(`<a href="https://njump.me/%s">%s (%s…%s)</a>`, nip19, name, firstChars, lastChars)
			} else {
				return fmt.Sprintf(`<a href="https://njump.me/%s">%s…%s</a>`, nip19, firstChars, lastChars)
			}
		} else {
			return fmt.Sprintf(`<a href="https://njump.me/%s">%s…%s</a>`, nip19, firstChars, lastChars)
		}
	})
}
