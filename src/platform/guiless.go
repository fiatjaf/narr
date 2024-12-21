//go:build !windows && !macos
// +build !windows,!macos

package platform

import (
	"github.com/fiatjaf/narr/src/server"
)

func Start(s *server.Server) {
	s.Start()
}
