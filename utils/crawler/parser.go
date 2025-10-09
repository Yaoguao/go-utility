package crawler

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

func ProcessHTMLAndExtractLinks(htmlBytes []byte, base *url.URL, c *Crawler) ([]byte, []string, error) {
	doc, err := html.Parse(bytes.NewReader(htmlBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("html parse error: %w", err)
	}

	imgDir := filepath.Join(c.Output, "assets", "images")
	cssDir := filepath.Join(c.Output, "assets", "css")
	jsDir := filepath.Join(c.Output, "assets", "js")
	_ = os.MkdirAll(imgDir, os.ModePerm)
	_ = os.MkdirAll(cssDir, os.ModePerm)
	_ = os.MkdirAll(jsDir, os.ModePerm)

	var links []string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "img", "source":
				handleAttrs := []string{"src", "srcset", "data-src", "data-srcset"}
				for ai := range n.Attr {
					key := n.Attr[ai].Key
					for _, h := range handleAttrs {
						if key == h {
							orig := n.Attr[ai].Val
							if h == "srcset" || h == "data-srcset" {
								newVal := rewriteSrcset(orig, base, c, "images", imgDir)
								if newVal != "" {
									n.Attr[ai].Val = newVal
								}
							} else {
								if newRel, err := handleResourceAndReturnRel(orig, base, c, "images", imgDir); err == nil && newRel != "" {
									n.Attr[ai].Val = newRel
								}
							}
						}
					}
				}
			case "script":
				for ai := range n.Attr {
					if n.Attr[ai].Key == "src" {
						orig := n.Attr[ai].Val
						if newRel, err := handleResourceAndReturnRel(orig, base, c, "js", jsDir); err == nil && newRel != "" {
							n.Attr[ai].Val = newRel
						}
					}
				}
			case "link":
				var hrefIdx = -1
				var relVal string
				for i := range n.Attr {
					if n.Attr[i].Key == "href" {
						hrefIdx = i
					}
					if n.Attr[i].Key == "rel" {
						relVal = n.Attr[i].Val
					}
				}
				if hrefIdx != -1 && strings.Contains(strings.ToLower(relVal), "stylesheet") {
					orig := n.Attr[hrefIdx].Val
					if newRel, err := handleResourceAndReturnRel(orig, base, c, "css", cssDir); err == nil && newRel != "" {
						n.Attr[hrefIdx].Val = newRel
					}
				}
			case "a":
				for i := range n.Attr {
					if n.Attr[i].Key == "href" {
						href := strings.TrimSpace(n.Attr[i].Val)
						if href != "" && !strings.HasPrefix(href, "javascript:") && !strings.HasPrefix(href, "mailto:") && !strings.HasPrefix(href, "#") {
							links = append(links, href)
						}
					}
				}
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch)
		}
	}
	walk(doc)

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return nil, nil, fmt.Errorf("render html error: %w", err)
	}
	return buf.Bytes(), links, nil
}

func rewriteSrcset(srcset string, base *url.URL, c *Crawler, resType, outDir string) string {
	parts := strings.Split(srcset, ",")
	var outParts []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		fields := strings.Fields(p)
		if len(fields) == 0 {
			continue
		}

		urlPart := fields[0]
		desc := ""
		if len(fields) > 1 {
			desc = strings.Join(fields[1:], " ")
		}
		local, err := handleResourceAndReturnRel(urlPart, base, c, resType, outDir)
		if err != nil || local == "" {
			if desc != "" {
				outParts = append(outParts, urlPart+" "+desc)
			} else {
				outParts = append(outParts, urlPart)
			}
			continue
		}
		if desc != "" {
			outParts = append(outParts, local+" "+desc)
		} else {
			outParts = append(outParts, local)
		}
	}
	return strings.Join(outParts, ", ")
}

func handleResourceAndReturnRel(attrVal string, base *url.URL, c *Crawler, resType string, outDir string) (string, error) {
	attrVal = strings.TrimSpace(attrVal)
	if attrVal == "" || strings.HasPrefix(attrVal, "data:") {
		return "", nil
	}
	parsed, err := url.Parse(attrVal)
	if err != nil {
		return "", err
	}
	abs := base.ResolveReference(parsed)
	absStr := abs.String()

	if rel, ok := c.ResourceMap[absStr]; ok {
		return rel, nil
	}

	data, err := DownloadToBytes(absStr)
	if err != nil {
		return "", err
	}

	ext := path.Ext(abs.Path)
	localName := path.Base(abs.Path)
	if localName == "" || localName == "/" || localName == "." {
		if ext == "" {
			ext = guessExtByType(resType)
		}
		localName = uniqueNameFromURL(absStr, ext)
	}
	localName = sanitizeFilename(localName)
	localPath := filepath.Join(outDir, localName)

	if err := SaveFile(localPath, data); err != nil {
		return "", err
	}

	rel := filepath.ToSlash(filepath.Join("assets", resType, localName))
	c.ResourceMap[absStr] = rel
	return rel, nil
}

func guessExtByType(resType string) string {
	switch resType {
	case "images":
		return ".img"
	case "css":
		return ".css"
	case "js":
		return ".js"
	default:
		return ""
	}
}

func sanitizeFilename(name string) string {
	if idx := strings.Index(name, "?"); idx != -1 {
		name = name[:idx]
	}
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	return name
}
