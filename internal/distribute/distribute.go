package distribute

import (
	"github.com/4nte/protodist/git"
	"github.com/4nte/protodist/internal/target"
	"io/ioutil"
	"log"
	"os"
	"path"
)

// Distribute proto to files
func Distribute(gitCfg git.Config, protoOutDir string) {
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
	if refType, refValue := gitCfg.ParseRef(); refType == "branch" {
		cloneBranch = refValue
	}
	target.Golang(protoOutDir, gitCfg, cloneBranch, cloneDir)
	target.Javascript(protoOutDir, gitCfg, cloneBranch, cloneDir)
	target.C(protoOutDir, gitCfg, cloneBranch, cloneDir)
}
