# Testing

In order to use these commands, you must export a valid Splunk HEC token and URL in your shell session.

```shell
export SPLUNK_HEC_TOKEN=<TOKEN HERE>
export SPLUNK_HEC_ENDPOINT=https://http-inputs-FOOBAR.splunkcloud.com
```

## Splunk HEC via cURL

This cURL command will submit a log event to the HEC collector. 

```shell
curl -v $SPLUNK_HEC_ENDPOINT/services/collector -H "Authorization: Splunk $SPLUNK_HEC_TOKEN" -d '{"sourcetype": "httpevent", "event":"Hello, World!"}' 

## you should see a HTTP 200 response with a body like ...
## {"text":"Success","code":0} 
```

To perform a health check, but not create a log event:

```shell
curl -v $SPLUNK_HEC_ENDPOINT/services/collector/health -H "Authorization: Splunk $SPLUNK_HEC_TOKEN"

## you should see a HTTP 200 response with a body like ...
## {"text":"HEC is healthy","code":17}
```

## xcaddy: Using testing Caddyfile

Make sure to run these command, from the top level of the project (e.g. not this `testing` directory)

```shell
# this starts caddy using the module
xcaddy run --config testing/Caddyfile

# make a test request to the server
curl https://127.0.0.1:8080 -v
```
