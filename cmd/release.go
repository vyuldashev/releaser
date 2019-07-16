package cmd

import (
	"compress/flate"
	"fmt"
	"github.com/mholt/archiver"
	"github.com/spf13/cobra"
	"github.com/vyuldashev/releaser/internal/config"
	"github.com/xanzy/go-gitlab"
	"log"
	"strings"
)

var (
	version string
)

func NewRelease(c *config.Config) *cobra.Command {
	releaseCmd := &cobra.Command{
		Use: "release",
		Run: func(cmd *cobra.Command, args []string) {
			if !strings.HasPrefix(version, "v") {
				version = fmt.Sprintf("v%s", version)
			}

			git := gitlab.NewClient(nil, c.GitLab.Token)
			err := git.SetBaseURL(c.GitLab.URL)

			if err != nil {
				log.Fatal(err)
			}

			path := createArchive(c.Files, version)
			projectFile := uploadArchive(git, c.ProjectID, path)
			release := createRelease(git, c.ProjectID, version)
			linkArchive(git, c.GitLab.URL, c.ProjectID, release.TagName, projectFile)

			log.Println("Release created!")
		},
	}

	releaseCmd.PersistentFlags().StringVar(&version, "version", "v1.0.0-dev", "version")
	releaseCmd.PersistentFlags().StringVar(&version, "clean", "v1.0.0-dev", "version")

	return releaseCmd
}

func createArchive(files []string, version string) (path string) {
	p := fmt.Sprintf("%s.tar.gz", version)

	z := archiver.TarGz{
		CompressionLevel: flate.DefaultCompression,
		Tar: &archiver.Tar{
			MkdirAll:               true,
			ContinueOnError:        false,
			OverwriteExisting:      true,
			ImplicitTopLevelFolder: false,
		},
	}

	err := z.Archive(files, p)

	if err != nil {
		log.Fatal(err)
	}

	return p
}

func uploadArchive(g *gitlab.Client, projectID string, file string) *gitlab.ProjectFile {
	f, _, err := g.Projects.UploadFile(projectID, file)

	if err != nil {
		log.Fatal(err)
	}

	return f
}

func createRelease(g *gitlab.Client, projectID string, name string) *gitlab.Release {
	opts := &gitlab.CreateReleaseOptions{
		Name:        gitlab.String(name),
		TagName:     gitlab.String(name),
		Description: gitlab.String(name),
	}

	rel, _, err := g.Releases.CreateRelease(projectID, opts)

	if err != nil {
		log.Fatal(err)
	}

	return rel
}

func linkArchive(g *gitlab.Client, gitlabURL string, projectID string, tagName string, file *gitlab.ProjectFile) {
	_, _, err := g.ReleaseLinks.CreateReleaseLink(projectID, tagName, &gitlab.CreateReleaseLinkOptions{
		Name: gitlab.String(file.Alt),
		URL:  gitlab.String(fmt.Sprintf("%s/%s/%s", gitlabURL, projectID, strings.TrimLeft(file.URL, "/"))),
	})

	if err != nil {
		log.Println(err)
	}
}
