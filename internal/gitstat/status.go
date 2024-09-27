package gitstat

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
)

var (
	errEmptyOutput = errors.New("empty output")
	re             = regexp.MustCompile(`^(.*)-(\d+)-g([0-9,a-f]+)$`)
)

func execGit(dir string, arg ...string) (string, error) {
	cmd := exec.Command("git", arg...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return string(out), err
	}
	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		return "", errEmptyOutput
	}
	return string(out), nil
}

// Description contains the result of `git describe --long` command. It could be
// empty if there is no tag in the repository.
type Description struct {
	Tag               string
	AdditionalCommits int // number of additional commits after the last tag
	ShortHash         string
}

// Status contains the status of the git repository in the current directory.
type Status struct {
	Branch      string      // result of `git branch --show-current`
	Description Description // result of `git describe --long` command
	Hash        string      // result of `git rev-parse HEAD` command
	ShortHash   string      // result of `git rev-parse --short HEAD` command
	AuthorDate  string      // result of `git log -n1 --date=format:"%Y-%m-%dT%H:%M:%S" --format=%ad`
	Dirty       bool        // repo returns non-empty `git status --porcelain`
}

// Get returns the status of the git repository in the current directory.
func Get() (status *Status, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("git status failed: %w", err)
		}
	}()
	dir := "."
	dir, err = filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	s := &Status{}
	_, err = exec.Command("git", "rev-parse", "--is-inside-work-tree").Output()
	if err != nil {
		return nil, errors.New("not inside a git repository")
	}
	s.Branch, err = execGit(dir, "branch", "--show-current")
	if err != nil {
		s.Branch = ""
	}
	s.Hash, err = execGit(dir, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	s.ShortHash, err = execGit(dir, "rev-parse", "--short", "HEAD")
	if err != nil {
		return nil, err
	}
	s.AuthorDate, err = execGit(dir, "log", "-n1", "--date=format:%Y-%m-%dT%H:%M:%S", "--format=%ad")
	if err != nil {
		return nil, err
	}

	o, err := execGit(dir, "status", "--porcelain")
	if err != nil && !errors.Is(err, errEmptyOutput) {
		return nil, err
	}
	s.Dirty = !(o == "" || o == "\n" || o == "\r\n")
	o, err = execGit(dir, "describe", "--tags", "--long")
	if err != nil {
		return s, nil //nolint:nilerr,nolintlint // No error, just no description.
	}
	d, err := parseDescription(o)
	if err != nil {
		return s, err
	}
	s.Description = *d
	return s, nil
}

func parseDescription(s string) (*Description, error) {
	parts := re.FindStringSubmatch(s)
	if len(parts) != 4 { //nolint:mnd // 4 is the expected number of parts.
		return nil, errors.New("failed to parse `git describe` result")
	}
	n, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, err
	}
	return &Description{
		Tag:               parts[1],
		AdditionalCommits: n,
		ShortHash:         parts[3],
	}, nil
}
