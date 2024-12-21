//go:build macos || windows
// +build macos windows

package platform

import (
	"github.com/fiatjaf/narr/src/server"
	"github.com/fiatjaf/narr/src/systray"
)

func Start(s *server.Server) {
	systrayOnReady := func() {
		systray.SetIcon(Icon)

		menuOpen := systray.AddMenuItem("Open", "")
		systray.AddSeparator()
		menuQuit := systray.AddMenuItem("Quit", "")

		go func() {
			for {
				select {
				case <-menuOpen.ClickedCh:
					Open(s.GetAddr())
				case <-menuQuit.ClickedCh:
					systray.Quit()
				}
			}
		}()

		s.Start()
	}
	systray.Run(systrayOnReady, nil)
}
