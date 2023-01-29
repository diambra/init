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
	"net/url"
	"reflect"
	"syscall"
	"text/template"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/mitchellh/mapstructure"
)

type Sources map[string]string

func (s *Sources) Validate() error {
	for path, us := range *s {
		if path == "" {
			return fmt.Errorf("path for source %s is empty", us)
		}
		if us == "" {
			return fmt.Errorf("url for path %s is empty", path)
		}
		u, err := url.Parse(us)
		if err != nil {
			return fmt.Errorf("invalid url %s for path %s: %w", us, path, err)
		}
		switch u.Scheme {
		case "http", "https":
			// ok
		default:
			return fmt.Errorf("invalid url %s for path %s: only http and https are supported", us, path)
		}
	}
	return nil
}

type Initializer struct {
	sources Sources
	secrets map[string]string
}

type TemplateData struct {
	Secrets *map[string]string
}

func NewInitializer(sources, secrets map[string]string) (*Initializer, error) {
	init := &Initializer{
		sources: sources,
		secrets: secrets,
	}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: func(from, to reflect.Type, data interface{}) (interface{}, error) {
			if to.Kind() == reflect.String && from.Kind() == reflect.String {
				tmpl, err := template.New("manifest").Parse(data.(string))
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
	return init, nil
}

func NewInitializerFromStrings(sourcesStr, secretsStr string) (*Initializer, error) {
	var (
		secrets map[string]string
		sources map[string]string
	)

	if err := json.Unmarshal([]byte(sourcesStr), &sources); err != nil {
		return nil, fmt.Errorf("failed to parse sources: %w", err)
	}
	if secretsStr != "" {
		if err := json.Unmarshal([]byte(secretsStr), &secrets); err != nil {
			return nil, fmt.Errorf("failed to parse secrets: %w", err)
		}
	}

	return NewInitializer(sources, secrets)
}

func (i *Initializer) Init(logger log.Logger) error {
	oldmask := syscall.Umask(0077)
	defer syscall.Umask(oldmask)
	for path, source := range i.sources {
		u, err := url.Parse(source)
		if err != nil {
			return err
		}
		switch u.Scheme {
		case "http", "https":
			u.User = url.UserPassword(u.User.Username(), "xxx")
			level.Info(logger).Log("msg", "downloading", "path", path, "source", u.String())
			if err := downloadHTTP(path, source); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported scheme %q", u.Scheme)
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
