package cmd

import (
	"compress/flate"
	"errors"
	"fmt"
	"github.com/blang/semver"
	"github.com/mholt/archiver"
	"github.com/spf13/cobra"
	"github.com/vyuldashev/releaser/internal/config"
	"github.com/vyuldashev/releaser/internal/version"
	"github.com/xanzy/go-gitlab"
	"log"
	"strings"
	"time"
)

var (
	releaseVersion string
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

			releaseVersion, releaseTag, err := getVersion(git, c.ProjectID, version.Clean(releaseVersion))

			if err != nil {
				log.Fatal(err)
			}

			changelog, err := getChangelog(git, c.ProjectID, releaseVersion)

			if err != nil {
				log.Fatal(err)
			}

			path := createArchive(c.Files, releaseVersion)
			projectFile := uploadArchive(git, c.ProjectID, path)
			release := createRelease(git, c.ProjectID, releaseVersion, releaseTag, changelog)
			linkArchive(git, c.GitLab.URL, c.ProjectID, release.TagName, projectFile)

			log.Printf("Release %s created!", releaseVersion)
		},
	}

	releaseCmd.PersistentFlags().StringVar(&releaseVersion, "version", "", "release version")

	return releaseCmd
}

func getVersion(g *gitlab.Client, projectID string, releaseVersion string) (ver semver.Version, tag *gitlab.Tag, err error) {
	tags, _, err := g.Tags.ListTags(projectID, nil)

	if err != nil {
		log.Fatal(err)
	}

	if releaseVersion == "" {
		rv, err := semver.Make(version.Clean(tags[0].Name))

		if err != nil {
			log.Fatal(err)
		}

		return rv, tags[0], nil
	}

	for _, tag := range tags {
		if version.Clean(tag.Name) == releaseVersion {
			tv, err := semver.Make(releaseVersion)

			if err != nil {
				log.Fatal(err)
			}

			return tv, tag, nil
		}
	}

	return semver.MustParse("0.0.0"), nil, errors.New("release version tag not found")
}

func getPreviousVersion(tags []*gitlab.Tag, releaseVersion semver.Version) (semver.Version, error) {
	// find previous tag in the same major version
	for _, tag := range tags {
		tagVersion, err := semver.Make(version.Clean(tag.Name))

		if err != nil {
			continue
		}

		if tagVersion.GE(releaseVersion) || tagVersion.Major != releaseVersion.Major {
			continue
		}

		return tagVersion, nil
	}

	return semver.MustParse("0.0.0"), errors.New("could not fetch previous version")
}

func getChangelog(g *gitlab.Client, projectID string, releaseVersion semver.Version) (string, error) {
	tags, _, err := g.Tags.ListTags(projectID, nil)

	if err != nil {
		log.Fatal(err)
	}

	previousVersion, err := getPreviousVersion(tags, releaseVersion)

	if err != nil {
		log.Fatal(err)
	}

	var releaseVersionTagDate *time.Time
	var previousVersionTagDate *time.Time

	for _, tag := range tags {
		if version.Clean(tag.Name) == previousVersion.String() {
			previousVersionTagDate = tag.Commit.CreatedAt
		}

		if version.Clean(tag.Name) == releaseVersion.String() {
			releaseVersionTagDate = tag.Commit.CreatedAt
		}
	}

	if releaseVersionTagDate == nil {
		return "", errors.New("failed to fetch release version tag date")
	}

	if previousVersionTagDate == nil {
		return "", errors.New("failed to fetch previous version tag date")
	}

	mrs, _, err := g.MergeRequests.ListMergeRequests(&gitlab.ListMergeRequestsOptions{
		State:         gitlab.String("merged"),
		CreatedAfter:  previousVersionTagDate,
		CreatedBefore: releaseVersionTagDate,
	})

	if err != nil {
		log.Fatal(err)
	}

	var changelog strings.Builder

	changelog.WriteString(fmt.Sprintf("### Release notes for %s\n", releaseVersion))
	changelog.WriteString("#### Merged Merge Requests\n")

	for _, mr := range mrs {
		changelog.WriteString(fmt.Sprintf("- %s [#%d](%s) ([%s](%s))\n", mr.Title, mr.ID, mr.WebURL, mr.Author.Username, mr.Author.WebURL))
	}

	return changelog.String(), nil
}

func createArchive(files []string, releaseVersion semver.Version) (path string) {
	p := fmt.Sprintf("%s.tar.gz", releaseVersion)

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

func createRelease(g *gitlab.Client, projectID string, releaseVersion semver.Version, releaseTag *gitlab.Tag, changelog string) *gitlab.Release {
	v := releaseVersion.String()

	opts := &gitlab.CreateReleaseOptions{
		Name:        gitlab.String(v),
		TagName:     gitlab.String(releaseTag.Name),
		Description: gitlab.String(changelog),
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
