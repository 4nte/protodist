package distribute

import (
	"fmt"
	"github.com/4nte/protodist/git"
	"github.com/4nte/protodist/internal/target"
	"io/ioutil"
	"log"
	"os"
	"path"
)

// Distribute proto to files
func Distribute(gitCfg git.Config, protoOutDir string, dryRun bool, deployTarget string, deployDir string) {
	if dryRun {
		fmt.Println("Dry run. Changes won't be pushed to GIT.")
	}

	cloneDir, err := ioutil.TempDir(os.TempDir(), "proto-git-clone-*")
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
	// master branch will be cloned by default
	cloneBranch := "master"

	// if ref is a branch, then the new branch will be created or checked out with the same branch name of the ref
	if deployTarget == "git" {
		if refType, refValue := gitCfg.ParseRef(); refType == "branch" {
			cloneBranch = refValue
		}
	}

	if deployTarget == "local" {
		target.Golang(protoOutDir, gitCfg, cloneBranch, cloneDir, dryRun, deployTarget, deployDir)
	} else {
		target.Golang(protoOutDir, gitCfg, cloneBranch, cloneDir, dryRun, deployTarget, deployDir)
		target.Javascript(protoOutDir, gitCfg, cloneBranch, cloneDir, dryRun, deployTarget, deployDir)
		target.C(protoOutDir, gitCfg, cloneBranch, cloneDir, dryRun, deployTarget, deployDir)
	}

}
