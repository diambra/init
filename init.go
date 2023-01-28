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
	"encoding/json"
	"fmt"
	"net/url"
	"syscall"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type Initializer struct {
	logger  log.Logger
	Sources map[string]string `json:"sources"`
}

func NewInitializerFromString(logger log.Logger, str string) (*Initializer, error) {
	sources := map[string]string{}
	if err := json.Unmarshal([]byte(str), &sources); err != nil {
		return nil, err
	}

	return &Initializer{
		logger:  logger,
		Sources: sources,
	}, nil
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
