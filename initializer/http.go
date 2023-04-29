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
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type Downloader interface {
	Download(path, source string) error
}

type httpDownloader struct {
	HTTPClient *http.Client
	root       string
}

func (d *httpDownloader) Download(path, source string) error {
	path = filepath.Join(d.root, path)
	resp, err := d.HTTPClient.Get(source)
	if err != nil {
		return err
	}
	if resp.StatusCode > 299 || resp.StatusCode < 200 {
		errBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unexpected status code %d", resp.StatusCode)
		}
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, errBody)
	}
	defer resp.Body.Close()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(path), err)
	}
	fh, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	_, err = io.Copy(fh, resp.Body)
	return err
}
