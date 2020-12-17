package git

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

// Config
type Config struct {
	Owner string
	Host  string
	Ref   string
	Token string
}

func NewConfig(repoOwner, host, ref, token string) (Config, error) {
	// Check if ref type is correct
	if !(strings.HasPrefix(ref, "refs/heads") || strings.HasPrefix(ref, "refs/tags")) {
		return Config{}, errors.New("git ref should be in format of refs/heads/* or refs/tags/* ")
	}

	return Config{
		Owner: repoOwner,
		Host:  host,
		Ref:   ref,
		Token: token,
	}, nil
}

type RefType string

const BranchRef RefType = "branch"
const tagRef RefType = "tag"

func (c Config) ParseRef() (RefType, string) {
	if strings.HasPrefix(c.Ref, "refs/heads/") {
		return BranchRef, strings.TrimPrefix(c.Ref, "refs/heads/")
	}

	if strings.HasPrefix(c.Ref, "refs/tags/") {
		return tagRef, strings.TrimPrefix(c.Ref, "refs/tags/")
	}

	panic(fmt.Sprintf("unable to parse ref: %s", c.Ref))
}

// Resolve repo URL from repo name
func (c Config) GetRepoURL(repoName string) string {
	if len(c.Token) > 0 {
		return fmt.Sprintf("https://%s:x-oauth-basic@%s/%s/%s.git", c.Token, c.Host, c.Owner, repoName)
	}

	transport := "git@" + c.Host
	return fmt.Sprintf("%s:%s", transport, path.Join(c.Owner, repoName))
}

func (c Config) GitBase() string {
	return path.Join(c.Host, c.Owner)
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
