package main

import (
	"fmt"
	"github.com/logrusorgru/aurora"
	"golang.org/x/net/publicsuffix"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Template-literal placeholders like ${encodeURIComponent(e)} are stripped
// before the parens check in looksLikeURL so legitimate templated URLs
// aren't mistaken for in-the-middle-of-an-expression false positives.
var templatePlaceholder = regexp.MustCompile(`\$\{[^}]*\}`)

// looksLikeURL filters out captured "endpoints" that are clearly not URLs —
// most commonly regex misfires that grab a fragment of minified JS like
// ".concat(JSON.stringify(e),". Real URL paths don't contain bare parens
// once template-literal interpolations are stripped.
func looksLikeURL(s string) bool {
	if s == "" {
		return false
	}
	stripped := templatePlaceholder.ReplaceAllString(s, "")
	return !strings.ContainsAny(stripped, "()")
}

// Helper Functions
func debugLog(debug bool, format string, v ...interface{}) {
	if debug {
		fmt.Printf(format, v...)
	}
}

func removeANSI(input string) string {
	return ansiEscape.ReplaceAllString(input, "")
}

func hasBundlePathPrefix(src string) bool {
	for _, p := range bundlePathPrefixes {
		if strings.Contains(src, p) {
			return true
		}
	}
	return false
}

func writeOutput(s string) {
	if outputFileWriter == nil {
		return
	}
	if _, err := outputFileWriter.WriteString(s + "\n"); err != nil {
		fmt.Fprintf(os.Stderr, "kneejerk: output write failed: %v\n", err)
		return
	}
	if err := outputFileWriter.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "kneejerk: output flush failed: %v\n", err)
	}
}

func determineSeverity(envVar string) string {
	upper := strings.ToUpper(envVar)
	awsAccessKey := strings.Contains(upper, "AWS") &&
		strings.Contains(upper, "ACCESS") &&
		(strings.Contains(upper, "ID") || strings.Contains(upper, "KEY"))
	if awsAccessKey || strings.Contains(upper, "SECRET") {
		return "high"
	}
	if strings.Contains(upper, "AWS") {
		return "medium"
	}
	if strings.Contains(upper, "API") &&
		(strings.Contains(upper, "URL") || strings.Contains(upper, "HOST") || strings.Contains(upper, "ROOT")) {
		return "low"
	}
	return "info"
}

func colorizeMessage(templateID string, outputType string, severity string, jsURL string, match string) (string, string) {
	templateIDColored := aurora.BrightGreen(templateID).String()
	outputTypeColored := aurora.BrightBlue(outputType).String()
	var severityColored string
	if severity == "high" {
		severityColored = aurora.Red(severity).String()
	} else if severity == "medium" {
		severityColored = aurora.Yellow(severity).String()
	} else if severity == "low" {
		severityColored = aurora.Green(severity).String()
	} else {
		severityColored = aurora.Blue(severity).String()
	}
	coloredMessage := fmt.Sprintf("[%s] [%s] [%s] %s [%s]", templateIDColored, outputTypeColored, severityColored, jsURL, match)
	uncoloredMessage := fmt.Sprintf("[%s] [%s] [%s] %s [%s]", templateID, outputType, severity, jsURL, match)
	return coloredMessage, uncoloredMessage
}

func urlJoin(baseURL string, relURL string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		// If baseURL parsing fails, return the original baseURL
		return baseURL
	}
	rel, err := url.Parse(relURL)
	if err != nil {
		// If relURL parsing fails, return the original baseURL
		return baseURL
	}
	return u.ResolveReference(rel).String()
}

func printAPI(debug bool, jsURL string, method string, endpoint string) {
	if len(method) > 12 {
		debugLog(debug, "Debug: Ignoring API path due to method length (possible false positive): [%s, %s]\n", method, endpoint)
		return
	}

	// Parse the jsURL and endpoint
	jsURLParsed, err := url.Parse(jsURL)
	if err != nil {
		debugLog(debug, "Debug: Failed to parse jsURL %s: %v\n", jsURL, err)
		return
	}
	endpointParsed, err := url.Parse(endpoint)

	// If endpoint parsed successfully and it's a full URL, check if the base domain matches
	if err == nil && endpointParsed.IsAbs() {
		if extractBaseDomain(jsURLParsed.Host) != extractBaseDomain(endpointParsed.Host) {
			return
		}
	}

	severity := determineSeverity(endpoint)

	coloredMessage, uncoloredMessage := colorizeMessage("kneejerk", "api", severity, jsURL, fmt.Sprintf(`"%s", "%s"`, method, endpoint))
	fmt.Println(coloredMessage)
	writeOutput(uncoloredMessage)
}

// extractBaseDomain returns the registrable domain (eTLD+1) for a hostname,
// so "api.example.co.uk" maps to "example.co.uk" rather than "co.uk". Ports
// are stripped, and IPs and single-label hosts are returned unchanged.
func extractBaseDomain(hostname string) string {
	host := strings.ToLower(strings.TrimSuffix(hostname, "."))
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if net.ParseIP(host) != nil {
		return host
	}
	if base, err := publicsuffix.EffectiveTLDPlusOne(host); err == nil {
		return base
	}
	return host
}

// normalizeDomain turns a -domain value like "https://Example.com/" or
// "*.example.com" into a bare lowercase domain.
func normalizeDomain(d string) string {
	d = strings.ToLower(strings.TrimSpace(d))
	if strings.Contains(d, "://") {
		if u, err := url.Parse(d); err == nil {
			d = u.Hostname()
		}
	}
	d = strings.TrimPrefix(d, "*.")
	return strings.Trim(d, ".")
}

var validHostnamePattern = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$`)

// Object-storage endpoints. A non-empty first capture group is the bucket
// name in virtual-hosted style; an empty one means path-style, where the
// bucket is the first path segment.
var cloudStoragePatterns = []*regexp.Regexp{
	regexp.MustCompile(`^(?:(.+)\.)?s3(?:[.-][a-z0-9-]+)*\.amazonaws\.com(?:\.cn)?$`),
	regexp.MustCompile(`^(?:(.+)\.)?storage\.googleapis\.com$`),
	regexp.MustCompile(`^()storage\.cloud\.google\.com$`),
	regexp.MustCompile(`^()firebasestorage\.googleapis\.com$`),
	regexp.MustCompile(`^(.+)\.firebasestorage\.app$`),
	regexp.MustCompile(`^(.+)\.blob\.core\.windows\.net$`),
	regexp.MustCompile(`^(.+)\.(?:[a-z0-9-]+\.)?digitaloceanspaces\.com$`),
	regexp.MustCompile(`^(.+)\.r2\.cloudflarestorage\.com$`),
	regexp.MustCompile(`^(.+)\.r2\.dev$`),
}

// Third-party hosts that appear in nearly every bundle (XML namespaces,
// React's error decoder) and carry no signal.
var noiseBaseDomains = map[string]struct{}{
	"w3.org":      {},
	"schema.org":  {},
	"reactjs.org": {},
	"react.dev":   {},
}

// bucketRef reports whether host is a cloud object-storage endpoint and, if
// so, returns a reference to the bucket: the host itself for virtual-hosted
// style, or host/bucket for path-style URLs when the path names one.
func bucketRef(host, firstPathSegment string) (string, bool) {
	for _, p := range cloudStoragePatterns {
		m := p.FindStringSubmatch(host)
		if m == nil {
			continue
		}
		if m[1] == "" && firstPathSegment != "" {
			return host + "/" + firstPathSegment, true
		}
		return host, true
	}
	return "", false
}

// classifyHost assigns a severity to a hostname found in JS. Hosts on the
// target domain are the interesting ones (sibling APIs, staging, admin
// panels); the apex itself is low since nearly every site references it.
// ok is false for hosts that shouldn't be reported at all: the page being
// scanned and known-noise third parties.
func classifyHost(host string, t scanTarget) (severity string, ok bool) {
	if host == t.host {
		return "", false
	}
	if t.domain != "" {
		if host == t.domain {
			return "low", true
		}
		if strings.HasSuffix(host, "."+t.domain) {
			return "high", true
		}
	}
	if _, noisy := noiseBaseDomains[extractBaseDomain(host)]; noisy {
		return "", false
	}
	return "info", true
}
