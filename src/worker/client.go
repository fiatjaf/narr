package worker

import (
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

type Client struct {
	httpClient *http.Client
	userAgent  string
}

func (c *Client) get(url string) (*http.Response, error) {
	return c.getConditional(url, "", "")
}

func (c *Client) getConditional(url, lastModified, etag string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	return c.httpClient.Do(req)
}

var client *Client

func SetVersion(num string) {
	client.userAgent = "Narr/" + num
}

var privateIPBlocks []*net.IPNet

func init() {
	for _, cidr := range []string{
		"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8",
		"169.254.0.0/16", "172.16.0.0/12", "192.0.0.0/24", "192.168.0.0/16",
		"::1/128", "fc00::/7", "fe80::/10",
	} {
		_, block, _ := net.ParseCIDR(cidr)
		privateIPBlocks = append(privateIPBlocks, block)
	}
}

// IsPrivateIP reports whether ip falls in a loopback, private, or
// link-local range and should not be reachable from feed/favicon fetches.
func IsPrivateIP(ip net.IP) bool {
	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

// safeDialControl rejects connections to private/loopback/link-local
// addresses at dial time, once DNS resolution has already happened. Because
// the address checked here is the exact one being connected to, this closes
// the DNS-rebinding TOCTOU gap a pre-resolution check would have, and it
// applies uniformly to every fetch (feed refresh, favicon discovery, add-feed,
// readability) and every redirect hop, since they all dial through this client.
func safeDialControl(network, address string, c syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("refusing to connect: could not parse address %q", host)
	}
	if IsPrivateIP(ip) {
		return fmt.Errorf("refusing to connect to private address %s", ip)
	}
	return nil
}

func init() {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
			Control: safeDialControl,
		}).DialContext,
		DisableKeepAlives:   true,
		TLSHandshakeTimeout: time.Second * 10,
	}
	httpClient := &http.Client{
		Timeout:   time.Second * 30,
		Transport: transport,
	}
	client = &Client{
		httpClient: httpClient,
		userAgent:  "Narr/1.0",
	}
}
