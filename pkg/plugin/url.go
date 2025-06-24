package plugin

import "net/url"

const (
	urlBase = "v1/api"
)

func renderURL(u *url.URL, upath string) string {
	u = u.JoinPath(urlBase, upath)
	return u.String()
}
