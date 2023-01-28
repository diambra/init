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

package main

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

type Initializer struct {
	logger  log.Logger
	Sources map[string]string
}

type TemplateData struct {
	Secrets *map[string]interface{}
}

func NewInitializerFromStrings(logger log.Logger, sourcesStr, secretsStr string) (*Initializer, error) {
	var (
		secrets map[string]interface{}
		sources map[string]interface{}
		init    = &Initializer{
			logger: logger,
		}
	)

	if err := json.Unmarshal([]byte(sourcesStr), &sources); err != nil {
		return nil, fmt.Errorf("failed to parse sources: %w", err)
	}
	if secretsStr != "" {
		if err := json.Unmarshal([]byte(secretsStr), &secrets); err != nil {
			return nil, fmt.Errorf("failed to parse secrets: %w", err)
		}
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
		Result: &init.Sources,
	})
	if err != nil {
		return nil, err
	}

	if err := decoder.Decode(sources); err != nil {
		return nil, fmt.Errorf("failed to parse sources: %w", err)
	}

	return init, nil
}

func (i *Initializer) Init() error {
	oldmask := syscall.Umask(0077)
	defer syscall.Umask(oldmask)
	for path, source := range i.Sources {
		u, err := url.Parse(source)
		if err != nil {
			return err
		}
		switch u.Scheme {
		case "http", "https":
			u.User = url.UserPassword(u.User.Username(), "xxx")
			level.Info(i.logger).Log("msg", "downloading", "path", path, "source", u.String())
			if err := downloadHTTP(path, source); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported scheme %q", u.Scheme)
		}
	}
	return nil
}
