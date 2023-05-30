package initializer

import (
	"reflect"
	"strings"
	"testing"

	"github.com/go-kit/log"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type mockHTTPDownloader struct {
	root       string
	downloaded Sources
}

func (d *mockHTTPDownloader) Download(path, source string) error {
	d.downloaded[strings.TrimPrefix(path, d.root+"/")] = source
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
		assets      Sources
		expected    Sources
		expectedLog []string
		expectedErr string
	}{
		{
			name: "simple",
			sources: map[string]string{
				"foo": "http://foo",
				"bar": "http://bar",
			},
			expected: map[string]string{
				"foo": "http://foo",
				"bar": "http://bar",
			},
			expectedLog: []string{
				"level=info msg=downloading path=foo source=http://foo",
				"level=info msg=downloading path=bar source=http://bar",
			},
		},
		{
			name: "template token",
			sources: map[string]string{
				"foo": "http://{{ .Secrets.token }}@foo",
				"bar": "http://bar",
			},
			secrets: map[string]string{
				"token": "secret",
			},
			expected: map[string]string{
				"foo": "http://secret@foo",
				"bar": "http://bar",
			},
			expectedLog: []string{
				"level=info msg=downloading path=foo source=http://xxxxx@foo",
				"level=info msg=downloading path=bar source=http://bar",
			},
		},
		{
			name: "template user/pass",
			sources: map[string]string{
				"foo": "http://user:{{ .Secrets.pass }}@foo",
				"bar": "http://bar",
			},
			secrets: map[string]string{
				"pass": "joshua",
			},
			expected: map[string]string{
				"foo": "http://user:joshua@foo",
				"bar": "http://bar",
			},
			expectedLog: []string{
				"level=info msg=downloading path=foo source=http://user:xxxxx@foo",
				"level=info msg=downloading path=bar source=http://bar",
			},
		},
		{
			name: "template user",
			sources: map[string]string{
				"foo": "http://{{ .Secrets.pass }}@foo",
				"bar": "http://bar",
			},
			secrets: map[string]string{
				"pass": "joshua",
			},
			expected: map[string]string{
				"foo": "http://joshua@foo",
				"bar": "http://bar",
			},
			expectedLog: []string{
				"level=info msg=downloading path=foo source=http://xxxxx@foo",
				"level=info msg=downloading path=bar source=http://bar",
			},
		},
		{
			name: "template url parameter",
			sources: map[string]string{
				"foo": "http://foo/foo?token={{ .Secrets.token }}",
				"bar": "http://bar",
			},
			secrets: map[string]string{
				"token": "abcd",
			},
			expected: map[string]string{
				"foo": "http://foo/foo?token=abcd",
				"bar": "http://bar",
			},
			expectedLog: []string{
				"level=info msg=downloading path=foo source=\"http://foo/foo?token=xxxxx\"",
				"level=info msg=downloading path=bar source=http://bar",
			},
		},
		{
			name: "template without secret",
			sources: map[string]string{
				"foo": "http://foo/foo?token={{ .Secrets.token }}",
				"bar": "http://bar",
			},
			expectedErr: `failed to parse sources: 1 error(s) decoding:

* error decoding '[foo]': template: manifest:1:32: executing "manifest" at <.Secrets.token>: map has no entry for key "token"`,
		},
		{
			name: "invalid path",
			sources: map[string]string{
				"../foo": "http://foo",
			},
			expectedErr: `invalid path ../foo: needs to be an relative path`,
		},
		{
			name: "simple with assets",
			sources: map[string]string{
				"foo": "http://foo",
				"bar": "http://bar",
			},
			assets: map[string]string{
				"/outside/sources/baz": "http://baz",
			},
			expected: map[string]string{
				"outside/sources/baz": "http://baz",
				"foo":                 "http://foo",
				"bar":                 "http://bar",
			},
			expectedLog: []string{
				"level=debug msg=downloading path=/outside/sources/baz source=http://baz",
				"level=info msg=downloading path=foo source=http://foo",
				"level=info msg=downloading path=bar source=http://bar",
			},
		},
	} {
		root := "/sources"
		t.Run(tc.name, func(t *testing.T) {
			sourcesCopy := tc.sources.Copy()
			init, err := NewInitializer(tc.sources, tc.secrets, tc.assets, root)
			if tc.expectedErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error: %v", tc.expectedErr)
				} else if diff := cmp.Diff(tc.expectedErr, err.Error()); diff != "" {
					t.Errorf("error mismatch (-want +got):\n%s", diff)
				}
				return
			}

			init.HTTPDownloader = &mockHTTPDownloader{
				root:       root,
				downloaded: make(map[string]string),
			}

			err = init.Init(logger)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.sources, sourcesCopy, cmpopt); diff != "" {
				t.Errorf("initializer modified sources mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.expected, init.HTTPDownloader.(*mockHTTPDownloader).downloaded, cmpopt); diff != "" {
				t.Errorf("downloads mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.expectedLog, sw.slices, cmpopt); diff != "" {
				t.Errorf("log mismatch (-want +got):\n%s", diff)
				t.Error(sw.slices)
				t.Error(tc.expectedLog)
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

func TestNewInitializerFromStrings(t *testing.T) {
	var (
		initializerContentComparer = cmp.Comparer(func(x, y *Initializer) bool {
			return reflect.DeepEqual(x.sources, y.sources) &&
				reflect.DeepEqual(x.secrets, y.secrets) &&
				reflect.DeepEqual(x.assets, y.assets)
		})
	)

	for _, tc := range []struct {
		name        string
		sources     string
		secrets     string
		assets      string
		expected    *Initializer
		expectedErr string
	}{
		{
			name:     "simple",
			sources:  `{ "foo": "http://foo", "bar": "http://bar" }`,
			expected: &Initializer{sources: map[string]string{"foo": "http://foo", "bar": "http://bar"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			init, err := NewInitializerFromStrings(tc.sources, tc.secrets, tc.assets, "/sources")
			if tc.expectedErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error: %v", tc.expectedErr)
				} else if diff := cmp.Diff(tc.expectedErr, err.Error()); diff != "" {
					t.Errorf("error mismatch (-want +got):\n%s", diff)
				}
				return
			}

			if diff := cmp.Diff(tc.expected, init, initializerContentComparer); diff != "" {
				t.Errorf("initializer mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
