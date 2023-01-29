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
	"os"

	"github.com/diambra/init/initializer"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

func main() {
	var (
		logger = log.With(log.NewLogfmtLogger(os.Stderr), "caller", log.Caller(3))

		sources = os.Getenv("SOURCES")
	)
	if sources == "" {
		level.Info(logger).Log("msg", "SOURCES not set, exiting")
		os.Exit(0)
	}

	init, err := initializer.NewInitializerFromStrings(sources, os.Getenv("SECRETS"))
	if err != nil {
		level.Error(logger).Log("msg", err.Error())
		os.Exit(1)
	}
	if err := init.Init(logger); err != nil {
		level.Error(logger).Log("msg", err.Error())
		os.Exit(1)
	}
}
