package target

import (
	"fmt"
	"github.com/4nte/protodist/git"
	"github.com/4nte/protodist/util"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
)

func Golang(protoOutDir string, gitCfg git.Config, branch string, cloneDir string) {
	var goPackages []string
	// Scan compiled go packages
	files, err := ioutil.ReadDir(path.Join(protoOutDir, "go", gitCfg.GitBase()))
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		if !f.IsDir() {
			panic("file is not expected to be here, only directories")
		}
		goPackages = append(goPackages, f.Name())
		fmt.Println(f.Name())
	}
	//defer os.RemoveAll(cloneDir)

	// Clone go proto repos
	for _, pkg := range goPackages {
		repoUrl := gitCfg.GetRepoURL(pkg)
		git.Clone(repoUrl, branch)
	}

	// Delete all .go files in cloned go proto repos
	for _, pkg := range goPackages {
		repoDir := path.Join(cloneDir, pkg)
		pkgCloneDir, err := ioutil.ReadDir(repoDir)
		if err != nil {
			panic(err)
		}

		// Search for files with .go extension && delete them
		for _, file := range pkgCloneDir {
			if !file.Mode().IsRegular() || filepath.Ext(file.Name()) != ".go" {
				continue
			}
			// Delete .go file
			err := os.Remove(file.Name())
			if err != nil {
				panic(err)
			}
		}

		// Move generate .go files to cloned repo dir
		generatedPkgDir := path.Join(protoOutDir, "go", gitCfg.Host, gitCfg.Owner, pkg)
		err = util.CopyDirectory(generatedPkgDir, repoDir)
		if err != nil {
			panic(err)
		}

	}

	AddCommitTagPush(gitCfg, goPackages)
}

func AddCommitTagPush(cfg git.Config, repos []string) {
	for _, repo := range repos {
		git.AddAll(repo)
		git.Commit(repo, "add pb files")
		if cfg.Tag != "" {
			git.Tag(repo, cfg.Tag)
		}
		git.Push(repo, cfg.Branch)
	}
}

func Javascript(protoOutDir string, gitCfg git.Config, branch string, cloneDir string) {
	var tsPackages []string

	repoName := "proto-all-js"
	repoUrl := gitCfg.GetRepoURL(repoName)
	git.Clone(repoUrl, branch)

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
	AddCommitTagPush(gitCfg, []string{repoName})
}
