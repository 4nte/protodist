package target

import (
	"github.com/4nte/protodist/git"
)

func AddCommitTagPush(cfg git.Config, repos []string, dryRun bool) {
	for _, repo := range repos {
		git.AddAll(repo)
		git.Commit(repo, "add pb files")

		refType, refName := cfg.ParseRef()

		if refType == git.TagRef {
			// Create a git tag
			git.Tag(repo, refName)
		}
		if dryRun {
			// Skip git push
			return
		}

		git.Push(repo, refName)
	}
}
