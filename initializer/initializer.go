/*
 Copyright 2023 The DIAMBRA Authors

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      https://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package initializer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/mitchellh/mapstructure"
)

type Sources map[string]string

func (s *Sources) Copy() Sources {
	c := make(Sources)
	for k, v := range *s {
		c[k] = v
	}
	return c
}

// FIXME: Merge with the one in init
func (s *Sources) Validate() error {
	for path, us := range *s {
		if !filepath.IsLocal(path) {
			return fmt.Errorf("invalid path %s: needs to be an relative path", path)
		}
		if us == "" {
			return fmt.Errorf("url for path %s is empty", path)
		}
		u, processor, redactedURL, err := parseAndRedact(us)
		if err != nil {
			return fmt.Errorf("invalid url %s for path %s: %w", redactedURL, path, err)
		}
		switch u.Scheme {
		case "http", "https":
			switch processor {
			case "", "zip", "unzip":
				// ok
			default:
				return fmt.Errorf("invalid processor %s for path %s: only zip and unzip are supported", processor, path)
			}
		case "git":
			switch processor {
			case "https", "http":
				// ok
			default:
				return fmt.Errorf("invalid processor %s for path %s: only http(s) are supported", processor, path)
			}
		default:
			return fmt.Errorf("invalid url %s for path %s: only http(s) and git+http(s) are supported", redactedURL, path)
		}
	}
	return nil
}

type Initializer struct {
	logger         log.Logger
	HTTPDownloader Downloader
	GitDownloader  Downloader
	ZipProcessor   Processor
	sources        Sources
	secrets        map[string]string
	assets         Sources
	root           string
}

type TemplateData struct {
	Secrets *map[string]string
}

func NewInitializer(logger log.Logger, sources Sources, secrets, assets map[string]string, root string) (*Initializer, error) {
	init := &Initializer{
		logger:  logger,
		root:    root,
		sources: sources.Copy(),
		secrets: secrets,
		assets:  assets,
		HTTPDownloader: &httpDownloader{
			HTTPClient: http.DefaultClient,
		},
		GitDownloader: NewGitDownloader(logger),
		ZipProcessor:  &ZipProcessor{},
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: func(from, to reflect.Type, data interface{}) (interface{}, error) {
			if to.Kind() == reflect.String && from.Kind() == reflect.String {
				tmpl, err := template.New("manifest").Option("missingkey=error").Parse(data.(string))
				if err != nil {
					return "", err
				}
				var buf bytes.Buffer
				if err := tmpl.Execute(&buf, TemplateData{Secrets: &secrets}); err != nil {
					return "", err
				}
				return buf.String(), nil
			}
			return data, nil
		},
		Result: &init.sources,
	})
	if err != nil {
		return nil, err
	}
	if err := decoder.Decode(sources); err != nil {
		return nil, fmt.Errorf("failed to parse sources: %w", err)
	}

	return init, init.sources.Validate()
}

func NewInitializerFromStrings(logger log.Logger, sourcesStr, secretsStr, assetsStr, root string) (*Initializer, error) {
	var (
		secrets map[string]string
		sources map[string]string
		assets  map[string]string
	)

	if err := json.Unmarshal([]byte(sourcesStr), &sources); err != nil {
		return nil, fmt.Errorf("failed to parse sources: %w", err)
	}
	if secretsStr != "" {
		if err := json.Unmarshal([]byte(secretsStr), &secrets); err != nil {
			return nil, fmt.Errorf("failed to parse secrets: %w", err)
		}
	}
	if assetsStr != "" {
		if err := json.Unmarshal([]byte(assetsStr), &assets); err != nil {
			return nil, fmt.Errorf("failed to parse assets: %w", err)
		}
	}

	return NewInitializer(logger, sources, secrets, assets, root)
}

func (i *Initializer) init() error {
	if err := i.processSources(level.Info(i.logger), i.sources); err != nil {
		return err
	}
	if err := i.processSources(level.Debug(i.logger), i.assets); err != nil {
		return err
	}
	return nil
}

func (i *Initializer) processSources(logger log.Logger, sources Sources) error {
	for path, source := range sources {
		u, processor, redactedURL, err := parseAndRedact(source)
		if err != nil {
			return err
		}
		switch u.Scheme {
		case "http", "https":
			logger.Log("msg", "downloading", "path", path, "source", redactedURL)
			if err := i.HTTPDownloader.Download(filepath.Join(i.root, path), u.String()); err != nil {
				return err
			}
		case "git":
			logger.Log("msg", "cloning", "path", path, "source", redactedURL)
			u.Scheme = processor
			if err := i.GitDownloader.Download(filepath.Join(i.root, path), u.String()); err != nil {
				return err
			}

		default:
			return fmt.Errorf("unsupported scheme %q", u.Scheme)
		}
		switch processor {
		case "zip", "unzip":
			logger.Log("msg", "processing", "path", path, "processor", processor)
			if err := i.ZipProcessor.Process(path); err != nil {
				return err
			}
		}
	}
	return nil
}

func (i *Initializer) Validate() error {
	return i.sources.Validate()
}

func (i *Initializer) Sources() string {
	js, err := json.Marshal(i.sources)
	if err != nil {
		panic(err)
	}
	return string(js)
}

func (i *Initializer) Secrets() string {
	js, err := json.Marshal(i.secrets)
	if err != nil {
		panic(err)
	}
	return string(js)
}

const redactedPlaceholder = "xxxxx"

func parseAndRedact(s string) (*url.URL, string, string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, "", "", err
	}

	var (
		processor string
	)
	parts := strings.SplitN(u.Scheme, "+", 2)
	if len(parts) == 2 {
		u.Scheme = parts[0]
		processor = parts[1]
	}

	redactedURL := url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   u.Path,
		User:   u.User,
	}
	if u.User != nil {
		if _, ok := redactedURL.User.Password(); !ok {
			// no password, so redact the username
			redactedURL.User = url.User(redactedPlaceholder)
		} else {
			redactedURL.User = url.UserPassword(redactedURL.User.Username(), redactedPlaceholder)
		}
	}

	values := u.Query()
	for _, v := range values {
		for i := range v {
			v[i] = redactedPlaceholder
		}
	}
	redactedURL.RawQuery = values.Encode()
	return u, processor, redactedURL.String(), nil
}
