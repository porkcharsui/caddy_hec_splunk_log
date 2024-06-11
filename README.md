# caddy_splunk_hec_log

This Caddy module extends logging to support outputting events directly to the Splunk HEC endpoint via HTTP.

Inspired by [neodyme-labs/influx_log](https://github.com/neodyme-labs/influx_log) caddy module.

## Install

First, the [xcaddy](https://github.com/caddyserver/xcaddy) command:

```shell
$ go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest
```

Then build Caddy with this Go module plugged in. For example:

```shell
$ xcaddy build --with github.com/porkcharsui/caddy_splunk_hec_log=.
```

# Usage

Make sure to set the log encoder format to `json`. All below fields are required: 

* `url` - configures the Splunk HEC endpoint
* `token` - Splunk HEC `token` (e.g. example below uses `SPLUNK_HEC_TOKEN`, set via environmental variable)
* `flush_interval` - (optional; defaults to 10s) duration between bulk log events flushing to Splunk HEC  

During Caddy startup, this module verifies connectivity to the configured HEC health check endpoint and will terminate if the health check is unsuccessful.

If a flush to the HEC fails, this module re-buffers the events and re-attempts to flush them on the next interval. If Caddy is terminated with events still in the buffer, the buffer will be flushed one time before shutdown. If flushing fails during shutdown, log events are lost even since they will not reach the HEC.

This module can be configured via a `caddy.json` or a `Caddyfile`:

## Caddyfile
```
example.nuna.cloud {
	root * example
	file_server
	log {
		format json
		output splunk_hec_log {
			url https://http-inputs-FOOBAR.splunkcloud.com
			token {$SPLUNK_HEC_TOKEN}
			flush_interval 2s
		}
	}
}
```

# TODO

- [ ] handle edge cases where Splunk HEC endpoint is inaccessible and Caddy is being terminated (e.g. write hole) 


# Legal

Splunk® and Splunk® Cloud Platform are registered trademarks of Splunk Inc. in the United States and other countries. The use of the "Splunk" trademark is for descriptive purposes only and does not imply any affiliation with or endorsement by Splunk Inc.