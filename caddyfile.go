package caddy_splunk_hec_log

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

// UnmarshalCaddyfile populates the SplunkHECLog struct from Caddyfile tokens. Syntax:
//
//	splunk_hec_log {
//		url <url>
//		token <token>
//		flush_interval <duration>
//	}
func (l *SplunkHECLog) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	// Consumes the option name
	if !d.NextArg() {
		return d.ArgErr()
	}

	for nesting := d.Nesting(); d.NextBlock(nesting); {
		switch d.Val() {
		case "url":
			if !d.NextArg() {
				return d.ArgErr()
			}

			l.Url = d.Val()
		case "token":
			if !d.NextArg() {
				return d.ArgErr()
			}

			if l.Token != "" {
				return d.Errf("token has already been specified: %v", l.Token)
			}

			l.Token = d.Val()
		case "flush_interval":
			if !d.NextArg() {
				return d.ArgErr()
			}

			flush_interval, err := caddy.ParseDuration(d.Val())
			if err != nil {
				return d.Errf("invalid flush_interval duration '%s': %v", d.Val(), err)
			}
			l.FlushInterval = caddy.Duration(flush_interval)
		}
	}

	return nil
}
