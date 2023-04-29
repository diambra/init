package initializer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestZipProcessor(t *testing.T) {

	expectedContents := map[string]string{
		"test/foo":          "hello world\n",
		"test/bar":          "something else\n",
		"test/baz/bux/date": "Sa 29. Apr 13:22:23 CEST 2023\n",
	}
	zipfile, err := filepath.Abs("testdata/test.zip")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(zipfile, "test"); err != nil {
		t.Fatal(err)
	}

	processor := &ZipProcessor{}
	if err := processor.Process("test"); err != nil {
		t.Fatal(err)
	}
	for path, expected := range expectedContents {
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		buf := make([]byte, len(expected))
		if _, err := f.Read(buf); err != nil {
			t.Fatal(err)
		}
		if string(buf) != expected {
			t.Fatalf("unexpected content in %s: %s", path, string(buf))
		}
	}
	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if _, ok := expectedContents[path]; !ok {
			t.Fatalf("unexpected file %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
