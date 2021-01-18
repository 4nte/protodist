package target

import (
	"fmt"
	"github.com/4nte/protodist/git"
	"github.com/4nte/protodist/util"
	"io/ioutil"
	"log"
	"os"
	"path"
)

func Javascript(protoOutDir string, gitCfg git.Config, cloneBranch string, cloneDir string, dryRun bool) {
	var tsPackages []string

	repoName := "proto-all-js"
	repoUrl := gitCfg.GetRepoURL(repoName)
	git.Clone(repoUrl, cloneBranch)

	packageDirs, err := ioutil.ReadDir(path.Join(protoOutDir, "ts"))
	if err != nil {
		log.Fatal(err)
	}
	for _, pkgDir := range packageDirs {
		if !pkgDir.IsDir() {
			panic("file is not expected to be here, only directories")
		}
		tsPackages = append(tsPackages, pkgDir.Name())
	}

	// Copy generated pb files to repo dirs
	for _, pkg := range tsPackages {
		pkgTargetDir := path.Join(cloneDir, repoName, pkg)
		if err := os.MkdirAll(pkgTargetDir, 0700); err != nil {
			panic(fmt.Errorf("failed to create dir for package: %s", err))
		}
		err := util.CopyDirectory(path.Join(protoOutDir, "ts", pkg), path.Join(cloneDir, repoName, pkg))
		if err != nil {
			panic(err)
		}

	}

	// Add to GIT
	AddCommitTagPush(gitCfg, []string{repoName}, dryRun)
}
