package sitecontract_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestStoreDownloadsKeepLocalizedBadgesAndAvailabilityAcrossAllPages(t *testing.T) {
	repo := repositoryRoot(t)
	cache := createControlledGermanCache(t, repo)
	output := t.TempDir()
	build := exec.Command("make", "build", "SITE_TRANSLATIONS="+cache, "SITE_OUTPUT_DIR="+output)
	build.Dir = repo
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build store download fixture: %v\n%s", err, combined)
	}
	for _, route := range []string{"index.html", "de/index.html", "amp/index.html", "de/amp/index.html"} {
		t.Run(route, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(output, filepath.FromSlash(route)))
			if err != nil {
				t.Fatal(err)
			}
			document, err := html.Parse(strings.NewReader(string(data)))
			if err != nil {
				t.Fatal(err)
			}
			locale := "en"
			if strings.HasPrefix(route, "de/") {
				locale = "de"
			}
			var stores []string
			var note *html.Node
			walkPageNodes(document, func(node *html.Node) {
				if node.Type != html.ElementNode {
					return
				}
				if node.Data == "a" && attribute(node, "class") == "store-badge" {
					stores = append(stores, attribute(node, "href"))
					if attribute(node, "href") == "https://apps.apple.com/app/picfetch/id6817901220" {
						walkPageNodes(node, func(image *html.Node) {
							if image.Type != html.ElementNode || (image.Data != "img" && image.Data != "amp-img") {
								return
							}
							if !strings.HasSuffix(attribute(image, "src"), "badges/mac-app-store-"+locale+".svg") {
								t.Errorf("Mac App Store badge has incorrect locale: %s", attribute(image, "src"))
							}
							if attribute(image, "height") != "44" || attribute(image, "alt") != "Mac App Store" {
								t.Error("Mac App Store badge must retain legible height and accessible store name")
							}
						})
					}
				}
				if node.Data == "p" && attribute(node, "class") == "availability-note" {
					note = node
				}
			})
			if len(stores) != 2 || stores[0] != "https://apps.apple.com/app/picfetch/id6817901220" || !strings.HasPrefix(stores[1], "https://apps.microsoft.com/") {
				t.Fatalf("store badges must link to both stores with Apple first, got %v", stores)
			}
			if note == nil || !strings.Contains(normalizedNodeText(note), "France") || !strings.Contains(normalizedNodeText(note), "encryption") {
				t.Fatal("missing translated App Store availability notice")
			}
			previous := note.PrevSibling
			for previous != nil && previous.Type != html.ElementNode {
				previous = previous.PrevSibling
			}
			if previous == nil || previous.Data != "ul" || attribute(previous, "class") != "grid dl" {
				t.Fatal("availability notice must appear immediately below the download cards")
			}
		})
	}
}
