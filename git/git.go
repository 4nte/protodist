package git

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

// Config
type Config struct {
	Owner  string
	Host   string
	Branch string
	Tag    string
}

// Resolve repo URL from repo name
func (g Config) GetRepoURL(repoName string) string {
	transport := "git@github.com"
	return fmt.Sprintf("%s:%s", transport, path.Join(g.Owner, repoName))
}

func (g Config) GitBase() string {
	return path.Join(g.Host, g.Owner)
}

// Clone
func Clone(repoUrl, branch string) {
	repoUrlFragments := strings.Split(repoUrl, "/")
	repoName := repoUrlFragments[len(repoUrlFragments)-1]
	repoName = strings.TrimSuffix(repoName, ".git")

	// Clone
	cmd := exec.Command("git", "clone", repoUrl)
	cmd.Dir = os.TempDir()
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(fmt.Errorf("failed to clone: %s: %s", repoUrl, err))
	}

	// Checkout
	cmd = exec.Command("git", "checkout", "-b", branch)
	cmd.Dir = path.Join(os.TempDir(), repoName)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(fmt.Errorf("failed to checkout branch %s: %s", branch, err))
	}
}

func AddAll(repoName string) {
	repoDir := path.Join(os.TempDir(), repoName)
	_, err := os.Stat(repoDir)
	if err != nil {
		panic(fmt.Errorf("failed to stat repo dir: %s: %s", repoDir, err))
	}

	cmd := exec.Command("git", "add", "*")
	cmd.Dir = repoDir
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(fmt.Errorf("failed to add to staging: %s: %s", repoName, err))
	}
}

func Tag(repoName string, tag string) {
	repoDir := path.Join(os.TempDir(), repoName)
	cmd := exec.Command("git", "tag", tag)
	cmd.Dir = repoDir
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(fmt.Errorf("failed to tag: %s: %s", repoName, err))
	}
}

func Commit(repoName string, message string) {
	repoDir := path.Join(os.TempDir(), repoName)
	_, err := os.Stat(repoDir)
	if err != nil {
		panic(fmt.Errorf("failed to stat repo dir: %s: %s", repoDir, err))
	}

	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoDir
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(fmt.Errorf("failed to commit: %s: %s", repoName, err))
	}
}

func Push(repoName string, branch string) {
	repoDir := path.Join(os.TempDir(), repoName)
	_, err := os.Stat(repoDir)
	if err != nil {
		panic(fmt.Errorf("failed to stat repo dir: %s: %s", repoDir, err))
	}

	cmd := exec.Command("git", "push", "--force", "--tags", "--set-upstream", "origin", branch)
	cmd.Dir = repoDir
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(fmt.Errorf("failed to push: %s: %s", repoName, err))
	}
}
