package distribute

import (
	"github.com/4nte/protodist/git"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/4nte/protodist/internal/target"
)

// Distribute proto to files
func Distribute(gitCfg git.Config, protoOutDir string) {
	cloneDir, err := ioutil.TempDir("tmp", "proto-git-clone-*")
	if err != nil {
		log.Fatal(err)
	}

	err = os.Setenv("TMPDIR", path.Join(cloneDir))
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

	<-time.After(5 * time.Second)
}
