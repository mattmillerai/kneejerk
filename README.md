# Kneejerk

Kneejerk is a pentesting command line tool for scanning environment variables and other information from React websites.

## Features
* Scans JavaScript files of a provided URL for environment variables.
* Performs API endpoint enumeration throughout React codebase
* Extracts hostnames referenced in JS bundles and source maps, flagging the target's own subdomains (internal APIs, staging, admin) and cloud storage buckets
* Outputs found env variables and API endpoints to the console or to a specified file.

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
```angular2html
[kneejerk] [js] [info] https://app.example.com/static/js/2.chunk.js [NODE_ENV:"production"]
[kneejerk] [js] [info] https://app.example.com/static/js/2.chunk.js [REACT_APP_BUILD:"Production"]
[kneejerk] [js] [high] https://app.example.com/static/js/2.chunk.js [REACT_APP_SECRET:"SECRET"]
[kneejerk] [js] [high] https://app.example.com/static/js/2.chunk.js [REACT_APP_KEY:"KEY"]
[kneejerk] [js] [info] https://app.example.com/static/js/2.chunk.js [REACT_APP_API_HOST:"https://app.example.com"]
[kneejerk] [js] [info] https://app.example.com/static/js/2.chunk.js [REACT_APP_WEB_HOST:"WEB_HOST"]
[kneejerk] [js] [info] https://app.example.com/static/js/2.chunk.js [REACT_APP_VERSION:"VERSION"]
[kneejerk] [js] [high] https://app.example.com/static/js/2.chunk.js [REACT_APP_NOT_SECRET_CODE:"NOT_SECRET_CODE"]
[kneejerk] [js] [medium] https://app.example.com/static/js/2.chunk.js [REACT_APP_CLIENT_DATA_BUCKET_NAME:"client-bucket"]
[kneejerk] [js] [medium] https://app.example.com/static/js/2.chunk.js [REACT_APP_REGION:"us-east-2"]
```

#### Example Output w/ nuclei

```bash
[tech-detect:react] [http] [info] https://app.example.com
[kneejerk] [js] [info] https://app.example.com/static/js/2.chunk.js [NODE_ENV:"production"]
[kneejerk] [js] [info] https://app.example.com/static/js/2.chunk.js [REACT_APP_BUILD:"Production"]
[kneejerk] [js] [high] https://app.example.com/static/js/2.chunk.js [REACT_APP_SECRET:"SECRET"]
[kneejerk] [js] [high] https://app.example.com/static/js/2.chunk.js [REACT_APP_KEY:"KEY"]
[kneejerk] [js] [info] https://app.example.com/static/js/2.chunk.js [REACT_APP_API_HOST:"https://app.example.com"]
[kneejerk] [js] [info] https://app.example.com/static/js/2.chunk.js [REACT_APP_WEB_HOST:"WEB_HOST"]
[kneejerk] [js] [info] https://app.example.com/static/js/2.chunk.js [REACT_APP_VERSION:"VERSION"]
[kneejerk] [js] [high] https://app.example.com/static/js/2.chunk.js [REACT_APP_NOT_SECRET_CODE:"NOT_SECRET_CODE"]
[kneejerk] [js] [medium] https://app.example.com/static/js/2.chunk.js [REACT_APP_CLIENT_DATA_BUCKET_NAME:"client-bucket"]
[kneejerk] [js] [medium] https://app.example.com/static/js/2.chunk.js [REACT_APP_REGION:"us-east-2"]
[tech-detect:google-font-api] [http] [info] https://app.example.com
[tech-detect:nginx] [http] [info] https://app.example.com
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
```angular2html
[kneejerk] [domain] [high] https://www.example.com/static/js/main.js [staging-api.example.com]
[kneejerk] [bucket] [medium] https://www.example.com/static/js/main.js [example-uploads.s3.amazonaws.com]
[kneejerk] [domain] [info] https://www.example.com/static/js/main.js [js.stripe.com]
[kneejerk] [domain] [high] https://www.example.com/static/js/main.js.map [admin.example.com]
```

#### Flags/Options
```angular2html
Kneejerk - A tool for scanning environment variables in .js files

optional arguments:
  -cookie string
        Cookie header value to send (e.g. cf_clearance=... from a browser session)
  -debug
        Print debugging statements
  -domain string
        Target domain for classifying hostnames found in JS (default: inferred from each scanned URL)
  -k    Skip TLS certificate verification (insecure)
  -l string
        Path to a file containing a list of URLs to scan
  -o string
        Path to output file
  -u string
        URL of the website to scan
  -ua string
        User-Agent header to send
  -version
        Print the version and exit
```

## Contributing

I welcome contributions from the community! If you have any suggestions, bug reports, or ideas for improvement, feel free to open an issue or submit a pull request.

## Support the project

If you find this project helpful and would like to support its development, please consider donating:  
  
[![Buy me a coffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/yOd1JU9MQe)

## License

This project is licensed under the MIT License.
