package distribute

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"

	"github.com/4nte/protodist/git"
	"github.com/4nte/protodist/util"
)

// getTargets scans all template dirs contained in `target-templates/` and returns a list of their names
func getTargets() []string {
	var targets []string

	entries, err := ioutil.ReadDir("target-templates")
	if err != nil {
		log.Fatalf("failed to read 'target-templates' dir: %s", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		targets = append(targets, entry.Name())
	}

	return targets
}

func DoWork(gitCfg git.Config, dryRun bool) {
	cloneDir, err := ioutil.TempDir(os.TempDir(), "proto-repos")
	if err != nil {
		log.Fatal(err)
	}

	if dryRun {
		log.Println("Dry run. Changes won't be pushed to GIT.")
	}

	cloneDir, err = ioutil.TempDir(os.TempDir(), "proto-git-clone-*")
	if err != nil {
		log.Fatal(err)
	}

	// Handle this differently - this is not good design
	err = os.Setenv("TMPDIR", path.Join(cloneDir))
	if err != nil {
		panic(err)
	}

	err = os.Setenv("GIT_AUTHOR_NAME", "protodist")
	if err != nil {
		panic(err)
	}
	err = os.Setenv("GIT_AUTHOR_EMAIL", "email@example.com")
	if err != nil {
		panic(err)
	}

	// if ref is a branch, then the new branch will be created or checked out with the same branch name of the ref
	cloneBranch := "main"
	if refType, refValue := gitCfg.ParseRef(); refType == "branch" {
		cloneBranch = refValue
	}

	// For every target
	// 1. clone git repo
	// 2. delete all files (except .git)
	// 3. copy file contents from build/<target> to repo dir
	targets := getTargets()
	for _, target := range targets {
		log.Printf("processing target: %s", target)
		targetRepoName := fmt.Sprintf("proto-%s", target)
		repoDirPath := path.Join(cloneDir, targetRepoName)
		repoUrl := gitCfg.GetRepoURL(targetRepoName)
		git.Clone(repoUrl, cloneBranch)

		// Cleanup all files in repo, except .git
		cmd := exec.Command("rm", "-rf", "*") // rm will not delete the .git file
		cmd.Dir = repoDirPath
		if err := cmd.Run(); err != nil {
			log.Fatalf("failed to remove files from repo at path: %s", repoDirPath)
		}

		// Copy build dir to repo dir
		if err := util.CopyDirectory(path.Join("build", target), repoDirPath); err != nil {
			log.Fatalf("failed to copy template files: %s", err)
		}
	}

	// For every target
	// 1. git add .
	// 2. git commit
	// 3. git tag (if set)
	// 4. git push
	for _, target := range targets {
		targetRepoName := fmt.Sprintf("proto-%s", target)
		git.AddAll(targetRepoName)
		if commit, ok := git.Commit(targetRepoName, "add pb files"); ok {
			log.Printf("commit created: %s", commit.Hash)
		} else {
			log.Printf("no commit has been create, there is nothing to commit.")
		}

		refType, refName := gitCfg.ParseRef()
		if refType == git.TagRef {
			// Create a git tag
			git.Tag(targetRepoName, refName)
		}
		if !dryRun {
			git.Push(targetRepoName, refName)
		}
	}
}
