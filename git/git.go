package git

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
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

type CommitInfo struct {
	Timestamp time.Time
	Hash      string
}

func (c CommitInfo) ModulePseudoVersion() string {
	//layoutGit := "20060102150405" +

	gitTimestamp := fmt.Sprintf("%d%02d%02d%02d%02d%02d", c.Timestamp.Year(), c.Timestamp.Month(), c.Timestamp.Day(), c.Timestamp.Hour(), c.Timestamp.Minute(), c.Timestamp.Second())

	fmt.Println("timestamp", c.Timestamp.String())
	return fmt.Sprintf("v0.0.0-%s-%s", gitTimestamp, c.Hash)
}

type RefType string

const BranchRef RefType = "branch"
const TagRef RefType = "tag"

func (c Config) ParseRef() (RefType, string) {
	if strings.HasPrefix(c.Ref, "refs/heads/") {
		return BranchRef, strings.TrimPrefix(c.Ref, "refs/heads/")
	}

	if strings.HasPrefix(c.Ref, "refs/tags/") {
		return TagRef, strings.TrimPrefix(c.Ref, "refs/tags/")
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

	// If target branch is main, we can skip git switch.
	if branch == "main" {
		return
	}

	// Switch to non-main branch
	cmd = exec.Command("git", "checkout", "-B", branch)
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

	cmd := exec.Command("git", "add", ".")
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

func Commit(repoName string, message string) (CommitInfo, bool) {
	repoDir := path.Join(os.TempDir(), repoName)
	_, err := os.Stat(repoDir)
	if err != nil {
		panic(fmt.Errorf("failed to stat repo dir: %s: %s", repoDir, err))
	}

	// cmd := exec.Command("test", "-z", "\"$(git status --porcelain)\"")
	// cmd.Dir = repoDir
	// cmd.Stderr = os.Stderr
	// cmd.Stdout = os.Stdout
	// if err := cmd.Run(); err == nil {
	// 	log.Println("there is nothing to commit, working tree must be clean")
	// 	return CommitInfo{}, false
	// }

	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoDir
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		log.Printf("failed to git commit: %s: %s", repoName, err)
	}

	cmd = exec.Command("git", "log", "-1", `--format="%at-%h"`, `--abbrev=12`)
	cmd.Dir = repoDir
	cmd.Stderr = os.Stderr
	//if err := cmd.Run(); err != nil {
	//	panic(fmt.Errorf("failed to get last git info: %s: %s", repoName, err))
	//}
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	commitLog := strings.ReplaceAll(string(out), ":", "")
	commitLog = strings.ReplaceAll(commitLog, `"`, "")
	commitData := strings.Split(commitLog, "-")
	unix, err := strconv.ParseInt(commitData[0], 10, 64)
	if err != nil {
		panic(fmt.Sprintf("failed to parse unix timestamp from git log: %s", err))
	}
	return CommitInfo{
		Timestamp: time.Unix(unix, 0).UTC(),
		Hash:      commitData[1],
	}, true
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
