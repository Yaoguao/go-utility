package crawler

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

type Crawler struct {
	BaseURL      *url.URL
	Output       string
	MaxDepth     int
	PagesVisited map[string]bool
	ResourceMap  map[string]string
}

func New(rawURL, output string, depth int) *Crawler {
	u, _ := url.Parse(rawURL)
	return &Crawler{
		BaseURL:      u,
		Output:       output,
		MaxDepth:     depth,
		PagesVisited: make(map[string]bool),
		ResourceMap:  make(map[string]string),
	}
}

func (c *Crawler) Start() error {
	fmt.Println("Start:", c.BaseURL.String())
	return c.Crawl(c.BaseURL.String(), 0)
}

func (c *Crawler) Crawl(raw string, depth int) error {
	if depth > c.MaxDepth {
		return nil
	}
	if c.PagesVisited[raw] {
		return nil
	}

	fmt.Printf("Crawling (depth %d): %s\n", depth, raw)

	body, err := DownloadToBytes(raw)
	if err != nil {
		fmt.Printf("Download page error (%s): %v\n", raw, err)
		return nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return err
	}

	modified, links, err := ProcessHTMLAndExtractLinks(body, parsed, c)
	if err != nil {
		fmt.Printf("Process HTML error (%s): %v\n", raw, err)
		return nil
	}

	localPath := c.localPathForPage(parsed)
	if err := SaveFile(localPath, modified); err != nil {
		fmt.Printf("Save page error (%s): %v\n", localPath, err)
		return nil
	}

	c.PagesVisited[raw] = true

	for _, l := range links {
		u, err := url.Parse(l)
		if err != nil {
			continue
		}

		abs := parsed.ResolveReference(u)

		if abs.Host == c.BaseURL.Host {
			c.Crawl(abs.String(), depth+1)
		}
	}
	return nil
}

func (c *Crawler) localPathForPage(u *url.URL) string {
	cleanPath := u.Path
	if cleanPath == "" || cleanPath == "/" {
		return filepath.Join(c.Output, "index.html")
	}
	cleanPath = strings.TrimPrefix(cleanPath, "/")

	ext := filepath.Ext(cleanPath)
	if ext == "" {
		dir := filepath.Join(c.Output, filepath.FromSlash(cleanPath))
		return filepath.Join(dir, "index.html")
	}
	dir := filepath.Join(c.Output, filepath.Dir(cleanPath))
	return filepath.Join(dir, filepath.Base(cleanPath))
}
