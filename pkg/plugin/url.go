package plugin

import "net/url"

func renderURL(u *url.URL, upath string) string {
	u = u.JoinPath(urlBase, upath)
	return u.String()
}
