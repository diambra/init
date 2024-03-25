package initializer

import (
	"fmt"
	"io"
	"net/url"
	"os/exec"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// FIXME: The logWriter doesn't non-progress output well
type logWriter struct {
	logger log.Logger
}

func (l *logWriter) Write(p []byte) (n int, err error) {
	s := 0
	for i := 0; i < len(p); i++ {
		if p[i] == '\n' {
			l.logger.Log("msg", string(p[s:i]))
			s = i + 1
		}
	}

	if s < len(p) { // Check for remaining text after the last newline character
		l.logger.Log("msg", string(p[s:]))
	}

	return len(p), nil
}

type gitDownloader struct {
	progress io.Writer
}

func NewGitDownloader(logger log.Logger) Downloader {
	return &gitDownloader{
		progress: &logWriter{level.Info(logger)},
	}
}

func (g *gitDownloader) Download(path, urls string) error {
	u, err := url.Parse(urls)
	if err != nil {
		return err
	}

	values, err := url.ParseQuery(u.Fragment)
	if err != nil {
		return err
	}
	if len(values) > 1 {
		return fmt.Errorf("invalid fragment %s: only one ref is supported", u.Fragment)
	}

	ref := "main"
	for k, v := range values {
		if k != "ref" {
			return fmt.Errorf("invalid fragment %s: only ref is supported", k)
		}
		if len(v) != 1 {
			return fmt.Errorf("invalid fragment %s: only one ref is supported", k)
		}
		ref = v[0]
	}
	u.Fragment = ""

	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", ref, u.String(), path)
	cmd.Stdout = g.progress
	cmd.Stderr = g.progress
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("couldn't clone repository: %w", err)
	}
	return nil
}
