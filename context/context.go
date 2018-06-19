package context

import (
	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"

	"net/url"
	"path"
	"strings"
)

type Context interface {
	Store(name, value string) error
	Load(name string) (string, error)
}

func NewContext(dsn string) (Context, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, errors.Annotatef(err, "cannot parse context data source name: %s")
	}

	var ctx Context

	switch u.Scheme {
	case "redis":
		pathPrefix := strings.TrimPrefix(u.Path, "/")
		log.WithFields(log.Fields{
			"path": pathPrefix,
		}).Debug("Writing offset using Redis backend using path")
		ctx, err = NewRedisContext(u.Host, pathPrefix)
	case "file":
		log.Debug("Writing offset using File backend")
		fallthrough
	default:
		log.Debug("Writing offset using File backend (default)")
		// default context to file
		ctx = NewFSContext(path.Dir(u.Path), path.Base(u.Path))
	}

	return ctx, err
}
