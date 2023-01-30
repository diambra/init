package initializer

import (
	"strings"
	"testing"

	"github.com/go-kit/log"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type mockHTTPDownloader struct {
	downloaded Sources
}

func (d *mockHTTPDownloader) Download(path, source string) error {
	d.downloaded[path] = source
	return nil
}

type sliceWriter struct {
	slices []string
}

func (w *sliceWriter) Write(p []byte) (n int, err error) {
	w.slices = append(w.slices, strings.TrimSuffix(string(p), "\n"))
	return len(p), nil
}

func TestInitializer(t *testing.T) {
	var (
		sw     = &sliceWriter{}
		logger = log.NewLogfmtLogger(sw)

		cmpopt = cmpopts.SortSlices(func(a, b string) bool { return a < b })
	)
	for _, tc := range []struct {
		name        string
		sources     Sources
		secrets     map[string]string
		expected    Sources
		expectedLog []string
	}{
		{
			name: "simple",
			sources: map[string]string{
				"/foo": "http://foo",
				"/bar": "http://bar",
			},
			expected: map[string]string{
				"/foo": "http://foo",
				"/bar": "http://bar",
			},
			expectedLog: []string{
				"level=info msg=downloading path=/foo source=http://foo",
				"level=info msg=downloading path=/bar source=http://bar",
			},
		},
		{
			name: "template token",
			sources: map[string]string{
				"/foo": "http://{{ .Secrets.token }}@foo",
				"/bar": "http://bar",
			},
			secrets: map[string]string{
				"token": "secret",
			},
			expected: map[string]string{
				"/foo": "http://secret@foo",
				"/bar": "http://bar",
			},
			expectedLog: []string{
				"level=info msg=downloading path=/foo source=http://xxxxx@foo",
				"level=info msg=downloading path=/bar source=http://bar",
			},
		},
		{
			name: "template user/pass",
			sources: map[string]string{
				"/foo": "http://user:{{ .Secrets.pass }}@foo",
				"/bar": "http://bar",
			},
			secrets: map[string]string{
				"pass": "joshua",
			},
			expected: map[string]string{
				"/foo": "http://user:joshua@foo",
				"/bar": "http://bar",
			},
			expectedLog: []string{
				"level=info msg=downloading path=/foo source=http://user:xxxxx@foo",
				"level=info msg=downloading path=/bar source=http://bar",
			},
		},
		{
			name: "template user",
			sources: map[string]string{
				"/foo": "http://{{ .Secrets.pass }}@foo",
				"/bar": "http://bar",
			},
			secrets: map[string]string{
				"pass": "joshua",
			},
			expected: map[string]string{
				"/foo": "http://joshua@foo",
				"/bar": "http://bar",
			},
			expectedLog: []string{
				"level=info msg=downloading path=/foo source=http://xxxxx@foo",
				"level=info msg=downloading path=/bar source=http://bar",
			},
		},
		{
			name: "template url parameter",
			sources: map[string]string{
				"/foo": "http://foo/foo?token={{ .Secrets.token }}",
				"/bar": "http://bar",
			},
			secrets: map[string]string{
				"token": "abcd",
			},
			expected: map[string]string{
				"/foo": "http://foo/foo?token=abcd",
				"/bar": "http://bar",
			},
			expectedLog: []string{
				"level=info msg=downloading path=/foo source=\"http://foo/foo?token=xxxxx\"",
				"level=info msg=downloading path=/bar source=http://bar",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			init, err := NewInitializer(tc.sources, tc.secrets)
			if err != nil {
				t.Fatal(err)
			}
			init.HTTPDownloader = &mockHTTPDownloader{
				downloaded: make(map[string]string),
			}

			if diff := cmp.Diff(tc.expected, init.sources, cmpopt); diff != "" {
				t.Errorf("sources mismatch (-want +got):\n%s", diff)
			}

			if err := init.Init(logger); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.expected, init.HTTPDownloader.(*mockHTTPDownloader).downloaded, cmpopt); diff != "" {
				t.Errorf("downloads mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.expectedLog, sw.slices, cmpopt); diff != "" {
				t.Errorf("log mismatch (-want +got):\n%s", diff)
			}
			sw.slices = nil

			/*
				if diff := cmp.Diff(tc.expectedLog, buf.String()); diff != "" {
					t.Errorf("log mismatch (-want +got):\n%s", diff)
				}
				buf.Reset()*/
		})
	}
}
