package save

import (
	"errors"
	"net/http"
	resp "url-shortener/internal/lib/api"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/lib/random"
	"url-shortener/internal/storage"

	"log/slog"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type Request struct {
	URL   string `json:"url" validate:"required,url"`
	Alias string `json:"alias,omitempty"`
}

type Response struct {
	resp.Response
	Alias string `json:"alias,omitempty"`
}

// TODO: move to config
const aliasLength = 6

//go:generate go run github.com/vektra/mockery/v2@v2.36.0 --name=URLSaver

type URLSaver interface {
	SaveURL(urlToSave string, alias string) (int64, error)
}

func New(log *slog.Logger, urlSaver URLSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers/url/save.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			msg := "failed to decode request body"

			log.Error(msg, sl.Err(err))

			render.JSON(w, r, resp.Error(msg))

			return
		}

		log.Info("request body decoded", slog.Any("request", req))

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)

			log.Error("invalid_request", sl.Err(err))

			render.JSON(w, r, resp.ValidationError(validateErr))

			return
		}

		// TODO: case when generated alias already exists
		alias := req.Alias
		if alias == "" {
			alias = random.NewRandomString(aliasLength)
		}

		id, err := urlSaver.SaveURL(req.URL, alias)
		if errors.Is(err, storage.ErrUrlExists) {
			msg := "url already exists"

			log.Info(msg, slog.String("url", req.URL))

			render.JSON(w, r, resp.Error(msg))

			return
		}
		if err != nil {
			msg := "failed to save url"

			log.Error(msg, sl.Err(err))

			render.JSON(w, r, resp.Error(msg))

			return
		}

		log.Info("url saved", slog.Int64("id", id))

		render.JSON(w, r, Response{
			Response: resp.OK(),
			Alias:    alias,
		})
	}
}
