package svc

import (
	"github.com/asim/go-micro/v3/metadata"
	"github.com/cs3org/reva/pkg/token"
	"io"
	"net/http"
	"strings"

	"github.com/owncloud/ocis/ocis-pkg/log"
	"github.com/owncloud/ocis/ocis-pkg/service/grpc"

	"github.com/go-chi/chi"
	thumbnails "github.com/owncloud/ocis/thumbnails/pkg/proto/v0"
	"github.com/owncloud/ocis/webdav/pkg/config"
	thumbnail "github.com/owncloud/ocis/webdav/pkg/dav/thumbnails"
)

// Service defines the extension handlers.
type Service interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
	Thumbnail(http.ResponseWriter, *http.Request)
}

// NewService returns a service implementation for Service.
func NewService(opts ...Option) Service {
	options := newOptions(opts...)

	m := chi.NewMux()
	m.Use(options.Middleware...)

	svc := Webdav{
		config: options.Config,
		log:    options.Logger,
		mux:    m,
	}

	m.Route(options.Config.HTTP.Root, func(r chi.Router) {
		r.Get("/remote.php/dav/files/{user}/*", svc.Thumbnail)
		r.Get("/remote.php/dav/public-files/{token}/*", svc.Thumbnail)
	})

	return svc
}

// Webdav defines implements the business logic for Service.
type Webdav struct {
	config *config.Config
	log    log.Logger
	mux    *chi.Mux
}

// ServeHTTP implements the Service interface.
func (g Webdav) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mux.ServeHTTP(w, r)
}

// Thumbnail implements the Service interface.
func (g Webdav) Thumbnail(w http.ResponseWriter, r *http.Request) {
	tr, err := thumbnail.NewRequest(r)
	if err != nil {
		g.log.Error().Err(err).Msg("could not create Request")
		w.WriteHeader(http.StatusBadRequest)
		mustWrite(g.log, w, []byte(err.Error()))
		return
	}

	t := r.Header.Get("X-Access-Token")
	md := make(metadata.Metadata)
	md.Set(token.TokenHeader, t)
	ctx := metadata.NewContext(r.Context(), md)

	c := thumbnails.NewThumbnailService("com.owncloud.api.thumbnails", grpc.DefaultClient)
	rsp, err := c.GetThumbnail(ctx, &thumbnails.GetThumbnailRequest{
		Filepath:      strings.TrimLeft(tr.Filepath, "/"),
		ThumbnailType: extensionToFiletype(strings.TrimLeft(tr.Extension, ".")),
		Width:         int32(tr.Width),
		Height:        int32(tr.Height),
		Source:	&thumbnails.GetThumbnailRequest_Cs3Source{
			Cs3Source: &thumbnails.CS3Source{
				Path: "/home" + tr.Filepath,
			},
		},
	})
	if err != nil {
		g.log.Error().Err(err).Msg("could not get thumbnail")
		w.WriteHeader(http.StatusBadRequest)
		mustWrite(g.log, w, []byte(err.Error()))
		return
	}

	if len(rsp.Thumbnail) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", rsp.GetMimetype())
	w.WriteHeader(http.StatusOK)
	mustWrite(g.log, w, rsp.Thumbnail)
}

func extensionToFiletype(ext string) thumbnails.GetThumbnailRequest_FileType {
	ext = strings.ToUpper(ext)
	switch ext {
	case "JPG", "PNG":
		val := thumbnails.GetThumbnailRequest_FileType_value[ext]
		return thumbnails.GetThumbnailRequest_FileType(val)
	case "JPEG", "GIF":
		val := thumbnails.GetThumbnailRequest_FileType_value["JPG"]
		return thumbnails.GetThumbnailRequest_FileType(val)
	default:
		return thumbnails.GetThumbnailRequest_FileType(-1)
	}
}

func mustWrite(logger log.Logger, w io.Writer, val []byte) {
	if _, err := w.Write(val); err != nil {
		logger.Error().Err(err).Msg("could not write response")
	}
}
