package version

import "strings"

func Clean(v string) string {
	return strings.TrimLeft(v, "v")
}
