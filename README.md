# init
This implements an init container to download users sources and other assets required
during an agent run.

## Configuration
### `SOURCES`
- json map of strings
- key is a path relative to /sources and the value is the url to download the source from
- currently only http(s) is supported
- additionally a processor can be specified. Currently only `unzip` is supported. Example:
```
{ "data": "https+unzip://example.com/my-source.zip" }
```
### `SECRETS`
- json map of strings
- key is a name and value the value of the secret
- secrets can be refered to in `SOURCES`. Example:
```
SECRETS='{"password": "my-secret"}'
SOURCES='{"data": "https+unzip://user:{{ .Secrets.password }}@example.com/my-source.zip"}'
```

### Assets
- same as `SOURCES` but the path can be absolute
- used internally for additional assets

## Docker
### Build
```
docker build -t init .
```

### Run
Using git:
```bash
docker run --rm \
    -e SOURCES='{".": "git+https://discordianfish:{{.Secrets.gh_token}}@github.com/discordianfish/diambra-agent.git#ref=main"}' \
    -e SECRETS="{\"gh_token\": \"$(pass dev/github/pat)\"}" \
    -v /tmp/sources:/sources init
```