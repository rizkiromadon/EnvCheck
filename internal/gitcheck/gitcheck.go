// Package gitcheck detects whether a target file is tracked by Git. It is
// strictly read-only: it never stages, commits, modifies .gitignore, or
// alters the repository in any way. If git is unavailable or the path is
// not inside a repository, it reports that cleanly rather than erroring.
package gitcheck

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
)

// Status describes the git-tracking state of a file.
type Status struct {
	InRepo   bool
	Tracked  bool
	GitFound bool
}

// Check reports whether filePath is tracked by Git, using `git ls-files`.
// It resolves the working directory relative to the file so it works
// regardless of the caller's current directory.
func Check(filePath string) (Status, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return Status{}, err
	}
	dir := filepath.Dir(absPath)
	base := filepath.Base(absPath)

	if _, err := exec.LookPath("git"); err != nil {
		return Status{GitFound: false}, nil
	}

	revCmd := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree")
	out, err := revCmd.Output()
	if err != nil || strings.TrimSpace(string(out)) != "true" {
		return Status{GitFound: true, InRepo: false}, nil
	}

	lsCmd := exec.Command("git", "-C", dir, "ls-files", "--error-unmatch", base)
	if err := lsCmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return Status{GitFound: true, InRepo: true, Tracked: false}, nil
		}
		return Status{GitFound: true, InRepo: true}, err
	}

	return Status{GitFound: true, InRepo: true, Tracked: true}, nil
}
