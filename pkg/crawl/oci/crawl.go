package oci

import (
	"context"
	"fmt"
	"github.com/aquasecurity/vex-collector/pkg/crawl/git"
	"github.com/aquasecurity/vex-collector/pkg/crawl/vex"
	"github.com/aquasecurity/vex-collector/pkg/vexhub"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/package-url/packageurl-go"
	"log/slog"
)

const imageSourceLabel = "org.opencontainers.image.source"

type Crawler struct {
	rootDir string
}

func NewCrawler(rootDir string) *Crawler {
	return &Crawler{rootDir: rootDir}
}

func (c *Crawler) Crawl(ctx context.Context, pkg vexhub.Package) error {
	src := pkg.URL
	if src == "" {
		repoURL, err := c.detectSrc(pkg.PURL)
		if err != nil {
			return fmt.Errorf("failed to detect source: %w", err)
		}
		src = repoURL
	}
	if err := vex.CrawlPackage(ctx, c.rootDir, src, pkg.PURL); err != nil {
		return fmt.Errorf("failed to crawl package: %w", err)
	}
	return nil
}

func (c *Crawler) detectSrc(purl packageurl.PackageURL) (string, error) {
	qs := purl.Qualifiers.Map()
	repositoryURL, ok := qs["repository_url"]
	if !ok {
		return "", fmt.Errorf("repository_url not found in %s", purl.String())
	}
	tag, ok := qs["tag"]
	if !ok {
		tag = "latest"
	}

	refStr := repositoryURL + ":" + tag
	ref, err := name.ParseReference(refStr)
	if err != nil {
		return "", fmt.Errorf("parsing reference %q: %v", refStr, err)
	}

	img, err := remote.Image(ref)
	if err != nil {
		return "", fmt.Errorf("reading image %q: %v", refStr, err)
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return "", fmt.Errorf("reading config %q: %v", refStr, err)
	}

	src, ok := cfg.Config.Labels[imageSourceLabel]
	if !ok {
		return "", fmt.Errorf("%s not found in %s", imageSourceLabel, refStr)
	}
	slog.Info("Found an image label", slog.String("label", imageSourceLabel),
		slog.String("value", src))

	u, err := git.NormalizeURL(src)
	if err != nil {
		return "", fmt.Errorf("normalizing URL %q: %v", src, err)
	}

	return u.String(), nil
}
