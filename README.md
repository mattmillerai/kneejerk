<p align="center">
  <img src="https://readme-typing-svg.demolab.com?font=Fira+Code&weight=700&size=28&pause=1000&color=FF9900&center=true&vCenter=true&width=320&repeat=false&lines=kneejerk" alt="kneejerk" />
</p>

<div align="center">
<pre>
 _                 _         _   
| |___ _  ___ ___ (_)___ _ _| |__
| / / ' \/ -_) -_)| / -_) '_| / /
|_\_\_||_\___\___|/ \___|_| |_\_\
                |__/             
</pre>
</div>

<p align="center">
  <strong>Fast Env Var, API Endpoint &amp; Hostname Extraction from JavaScript Bundles for Pentesters &amp; Red Teamers</strong>
</p>

<p align="center">
  <a href="https://github.com/mattmillerai/kneejerk/releases/latest"><img src="https://img.shields.io/github/v/release/mattmillerai/kneejerk?color=ff9900&style=flat-square" alt="Latest Release"></a>
  <a href="https://github.com/mattmillerai/kneejerk/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/mattmillerai/kneejerk/ci.yml?label=tests&style=flat-square" alt="Tests"></a>
  <a href="https://github.com/mattmillerai/kneejerk/blob/main/LICENSE"><img src="https://img.shields.io/github/license/mattmillerai/kneejerk?style=flat-square" alt="License"></a>
  <img src="https://img.shields.io/github/go-mod/go-version/mattmillerai/kneejerk?style=flat-square&logo=go" alt="Go Version">
</p>

---

Kneejerk is a pentesting command line tool that scans a website's JavaScript bundles, and the source maps they reference, for hostnames, environment variables and API endpoints. Hostnames are classified against the target domain, flagging its own subdomains (internal APIs, staging, admin) and cloud storage buckets. Use `-cookie` to scan sites that sit behind a browser session or a Cloudflare challenge.

## Features
* Scans the JavaScript bundles a page loads (`.js`/`.mjs` under `/static/`, `/_next/static/`, `/assets/`, `/_app/immutable/` or `/_astro/`), which covers React (CRA), Next.js, Vite/Remix, SvelteKit and Astro builds.
* Extracts build-time environment variables (`NODE_`, `REACT_APP_`, `NEXT_PUBLIC_`, `VITE_`, `EXPO_PUBLIC_` and `AWS_` prefixes) and rates them by severity.
* Performs API endpoint enumeration throughout the bundles (axios, `fetch`, jQuery `ajax`, `XMLHttpRequest` and `"METHOD", "/path"` style calls).
* Extracts and classifies hostnames: the target's own subdomains (internal APIs, staging, admin), cloud storage buckets and third-party hosts.
* Follows `sourceMappingURL` comments and scans the source map's embedded sources too.
* Sends a `-cookie` header only to the scanned origin, so scans can run behind a browser session or Cloudflare challenge without leaking it to third-party hosts.
* Outputs findings to the console or to a specified file.

## Installation

### Homebrew (macOS/Linux)

```sh
brew install mattmillerai/tap/kneejerk
```

### Go Install

```sh
go install -v github.com/mattmillerai/kneejerk/cmd/kneejerk@latest
```

> Versions <= v0.3.1 were published under the module path `github.com/MillerMedia/kneejerk`. `go install .../kneejerk@latest` will reject that old path with a module path mismatch error — switch to `github.com/mattmillerai/kneejerk` as shown above.

## Usage

#### Single URL
```bash
kneejerk -u https://www.example.com -o output.txt
```

#### List of URLs
```bash
kneejerk -l urls.txt
```
`urls.txt` holds one URL per line.

#### Session cookie and self-signed certificates
```bash
# Send a cookie (e.g. cf_clearance from a browser session) to the scanned site
kneejerk -u https://www.example.com -cookie 'cf_clearance=...'

# Skip TLS certificate verification (insecure); proxy settings from the environment are still honored
kneejerk -u https://internal.example.com -k
```
The cookie is only sent to the same origin (scheme, host and port) as the URL being scanned, never to third-party scripts or source maps, and is dropped on cross-origin redirects.

#### Version
```bash
kneejerk -version
```

#### Using with [nuclei](https://github.com/projectdiscovery/nuclei)
```bash
nuclei -u https://www.example.com | kneejerk
```

#### Chained with other [Project Discovery](https://github.com/projectdiscovery) tools
```bash
subfinder -d example.com | httpx | nuclei | kneejerk
```

#### Example Output
```text
[kneejerk] [env-var] [info] https://app.example.com/static/js/main.js [NODE_ENV:"production"]
[kneejerk] [env-var] [low] https://app.example.com/static/js/main.js [REACT_APP_API_HOST:"https://api.example.com"]
[kneejerk] [env-var] [high] https://app.example.com/static/js/main.js [REACT_APP_SECRET:"s3cr3t"]
[kneejerk] [env-var] [info] https://app.example.com/static/js/main.js [NEXT_PUBLIC_REGION:"us-east-2"]
[kneejerk] [env-var] [info] https://app.example.com/static/js/main.js [VITE_BUCKET:"client-bucket"]
[kneejerk] [api] [info] https://app.example.com/static/js/main.js ["GET", "/api/v1/users"]
[kneejerk] [api] [info] https://app.example.com/static/js/main.js ["POST", "/api/v1/login"]
[kneejerk] [domain] [high] https://app.example.com/static/js/main.js [api.example.com]
[kneejerk] [domain] [high] https://app.example.com/static/js/main.js [staging-api.example.com]
[kneejerk] [bucket] [medium] https://app.example.com/static/js/main.js [example-uploads.s3.amazonaws.com]
[kneejerk] [domain] [info] https://app.example.com/static/js/main.js [js.stripe.com]
[kneejerk] [api] [info] https://app.example.com/static/js/main.js.map ["DELETE", "/api/v1/admin/users"]
[kneejerk] [domain] [high] https://app.example.com/static/js/main.js.map [admin.example.com]
```

#### Example Output w/ nuclei

Reading from stdin expects [nuclei](https://github.com/projectdiscovery/nuclei)-style output lines, where the URL is the fourth space-separated field (`[template] [type] [severity] URL`). Each input line is echoed before its findings. A bare URL on stdin is not accepted (it prints `Invalid input:`), so use `-u` or `-l` for plain URLs.

```text
[tech-detect:react] [http] [info] https://app.example.com/
[kneejerk] [env-var] [info] https://app.example.com/static/js/main.js [NODE_ENV:"production"]
[kneejerk] [env-var] [low] https://app.example.com/static/js/main.js [REACT_APP_API_HOST:"https://api.example.com"]
[kneejerk] [env-var] [high] https://app.example.com/static/js/main.js [REACT_APP_SECRET:"s3cr3t"]
[kneejerk] [env-var] [info] https://app.example.com/static/js/main.js [NEXT_PUBLIC_REGION:"us-east-2"]
[kneejerk] [env-var] [info] https://app.example.com/static/js/main.js [VITE_BUCKET:"client-bucket"]
[kneejerk] [api] [info] https://app.example.com/static/js/main.js ["GET", "/api/v1/users"]
[kneejerk] [api] [info] https://app.example.com/static/js/main.js ["POST", "/api/v1/login"]
[kneejerk] [domain] [high] https://app.example.com/static/js/main.js [api.example.com]
[kneejerk] [domain] [high] https://app.example.com/static/js/main.js [staging-api.example.com]
[kneejerk] [bucket] [medium] https://app.example.com/static/js/main.js [example-uploads.s3.amazonaws.com]
[kneejerk] [domain] [info] https://app.example.com/static/js/main.js [js.stripe.com]
[kneejerk] [api] [info] https://app.example.com/static/js/main.js.map ["DELETE", "/api/v1/admin/users"]
[kneejerk] [domain] [high] https://app.example.com/static/js/main.js.map [admin.example.com]
```

#### Hostname extraction
Hostnames found in JS bundles and source-map `sourcesContent` are classified against the target domain, which is inferred from each scanned URL (`app.example.com` → `example.com`, `www.example.co.uk` → `example.co.uk`) or set with `-domain`:

| Severity | Type | Meaning |
|----------|------|---------|
| high | `domain` | Subdomain of the target (`staging-api.example.com`, `admin.example.com`) |
| medium | `bucket` | Cloud storage: S3, GCS, Azure Blob, DigitalOcean Spaces, R2, Firebase Storage, `s3://` / `gs://` URIs |
| low | `domain` | The target's apex domain itself |
| info | `domain` | Third-party hosts |

The scanned host itself and ubiquitous noise (`w3.org`, `reactjs.org`, ...) aren't reported. Third-party hosts are only picked up from absolute URLs; on-domain hosts are also found as bare strings.

```bash
kneejerk -u https://www.example.com -domain example.com
```
```text
[kneejerk] [domain] [high] https://www.example.com/static/js/main.js [staging-api.example.com]
[kneejerk] [bucket] [medium] https://www.example.com/static/js/main.js [example-uploads.s3.amazonaws.com]
[kneejerk] [domain] [info] https://www.example.com/static/js/main.js [js.stripe.com]
[kneejerk] [domain] [high] https://www.example.com/static/js/main.js.map [admin.example.com]
```

#### Flags/Options
```text
Usage of kneejerk:
  -cookie string
    	Cookie header value to send (e.g. cf_clearance=... from a browser session)
  -debug
    	Print debugging statements
  -domain string
    	Target domain for classifying hostnames found in JS (default: inferred from each scanned URL)
  -k	Skip TLS certificate verification (insecure)
  -l string
    	Path to a file containing a list of URLs to scan
  -o string
    	Path to output file
  -u string
    	URL of the website to scan
  -ua string
    	User-Agent header to send (default "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
  -version
    	Print the version and exit
```

## Contributing

I welcome contributions from the community! If you have any suggestions, bug reports, or ideas for improvement, feel free to open an issue or submit a pull request. See [CONTRIBUTING.md](CONTRIBUTING.md) for commit/PR title conventions and how releases are cut.

## Support the project

If you find this project helpful and would like to support its development, please consider donating:  
  
[![Buy me a coffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/yOd1JU9MQe)

## License

This project is licensed under the MIT License.
