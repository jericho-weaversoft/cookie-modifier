# Cookie Modifier Traefik Plugin

A Traefik middleware plugin that transforms cookie names and dynamically sets cookie domains based on the target service URL.

## Features

- Rename cookies (e.g., `flowise_token` → `simple_token`)
- Set dynamic cookie domains based on request host
- Handle both request and response cookies
- Configurable security attributes (Secure, HttpOnly, SameSite)
- Debug logging support

## Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `sourceCookieName` | string | `flowise_token` | Name of the cookie to transform |
| `targetCookieName` | string | `simple_token` | New name for the cookie |
| `useDynamicDomain` | bool | `true` | Set cookie domain to request host |
| `secure` | bool | `false` | Add Secure attribute to cookies |
| `httpOnly` | bool | `false` | Add HttpOnly attribute to cookies |
| `sameSite` | string | `Lax` | SameSite attribute value |
| `path` | string | `/` | Cookie path |
| `debug` | bool | `false` | Enable debug logging |

## Usage

### Static Configuration

```yaml
# traefik.yml
experimental:
  plugins:
    cookie-modifier:
      moduleName: "github.com/jericho-weaversoft/cookie-modifier"
      version: "v1.0.0"