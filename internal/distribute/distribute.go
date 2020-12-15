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
	branch := gitCfg.Branch

	cloneDir, err := ioutil.TempDir("tmp", "proto-git-clone-*")
	if err != nil {
		log.Fatal(err)
	}

	err = os.Setenv("TMPDIR", path.Join(cloneDir))
	if err != nil {
		panic(err)
	}
	target.Golang(protoOutDir, gitCfg, branch, cloneDir)
	target.Javascript(protoOutDir, gitCfg, branch, cloneDir)

	<-time.After(5 * time.Second)
}
