package engine

import "strings"

type Parser struct{}

// parsing maps a request URL path to a RingType by longest-prefix match.
// Falls back to Default if no prefix matches.
func (p *Parser) parsing(url string) RingType {
	for pfx, ringType := range prefix {
		if strings.HasPrefix(url, pfx) {
			return ringType
		}
	}
	return Default
}

var prefix = map[string]RingType{
	"/api": API,
}
