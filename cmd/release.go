package cmd

import (
	"compress/flate"
	"errors"
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
			git := gitlab.NewClient(nil, c.GitLab.Token)
			err := git.SetBaseURL(c.GitLab.URL)

			if err != nil {
				log.Fatal(err)
			}

			version, err := getVersion(git, c.ProjectID, version)

			if err != nil {
				log.Fatal(err)
			}

			path := createArchive(c.Files, version)
			projectFile := uploadArchive(git, c.ProjectID, path)
			release := createRelease(git, c.ProjectID, version)
			linkArchive(git, c.GitLab.URL, c.ProjectID, release.TagName, projectFile)

			log.Printf("Release %s created!", version)
		},
	}

	releaseCmd.PersistentFlags().StringVar(&version, "version", "", "version")

	return releaseCmd
}

func getVersion(g *gitlab.Client, projectID string, version string) (string, error) {
	tags, _, err := g.Tags.ListTags(projectID, nil)

	if err != nil {
		log.Fatal(err)
	}

	if version == "" {
		return tags[0].Name, nil
	}

	for _, tag := range tags {
		if tag.Name == version {
			return version, nil
		}
	}

	return "", errors.New("tag not found")
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
