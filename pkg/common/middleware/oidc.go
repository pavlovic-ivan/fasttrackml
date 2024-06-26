package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rotisserie/eris"
	log "github.com/sirupsen/logrus"

	"github.com/G-Research/fasttrackml/pkg/common/api"
	"github.com/G-Research/fasttrackml/pkg/common/auth"
	"github.com/G-Research/fasttrackml/pkg/common/dao/repositories"
)

// nolint:gosec
const (
	oidcUserContextKey = "oidc_user"
)

// OIDCMiddleware represents OIDC middleware.
type OIDCMiddleware struct {
	client          auth.OIDCClientProvider
	rolesRepository repositories.RoleRepositoryProvider
}

// NewOIDCMiddleware creates new OIDC middleware logic.
func NewOIDCMiddleware(
	client auth.OIDCClientProvider,
	rolesRepository repositories.RoleRepositoryProvider,
) fiber.Handler {
	return OIDCMiddleware{
		client:          client,
		rolesRepository: rolesRepository,
	}.Handle()
}

// Handle handles OIDC middleware logic.
func (m OIDCMiddleware) Handle() fiber.Handler {
	return func(ctx *fiber.Ctx) (err error) {
		switch {
		case AdminPrefixRegexp.MatchString(ctx.Path()):
			return m.handleAdminResourceRequest(ctx)
		case ChooserPrefixRegexp.MatchString(ctx.Path()):
			return m.handleChooserResourceRequest(ctx)
		case MlflowAimPrefixRegexp.MatchString(ctx.Path()):
			return m.handleAimMlflowResourceRequest(ctx)
		}
		return ctx.Next()
	}
}

// handleAdminResourceRequest applies OIDC check for Admin resources.
func (m OIDCMiddleware) handleAdminResourceRequest(ctx *fiber.Ctx) error {
	authToken := strings.Replace(ctx.Get("Authorization"), "Bearer ", "", 1)
	if authToken == "" {
		log.Error("auth token has incorrect format")
		return ctx.Redirect("/login", http.StatusMovedPermanently)
	}
	user, err := m.client.Verify(ctx.Context(), authToken)
	if err != nil {
		log.Errorf("error verifying access token: %+v", err)
		return ctx.Redirect("/login", http.StatusMovedPermanently)
	}

	log.Debugf("user has roles: %v accociated", user.GetRoles())
	if !user.IsAdmin() {
		return ctx.Redirect("/errors/not-found", http.StatusMovedPermanently)
	}
	return ctx.Next()
}

// handleChooserResourceRequest applies OIDC check for Chooser resources.
func (m OIDCMiddleware) handleChooserResourceRequest(ctx *fiber.Ctx) error {
	namespace, err := GetNamespaceFromContext(ctx.Context())
	if err != nil {
		return ctx.Redirect("/errors/not-found", http.StatusMovedPermanently)
	}
	log.Debugf("checking access permission to %s namespace", namespace.Code)

	if path := ctx.Path(); path != "/login" && !strings.Contains(path, "/chooser/static") {
		authToken := strings.Replace(ctx.Get("Authorization"), "Bearer ", "", 1)
		if authToken == "" {
			log.Error("auth token has incorrect format")
			return ctx.Redirect("/login", http.StatusMovedPermanently)
		}
		user, err := m.client.Verify(ctx.Context(), authToken)
		if err != nil {
			log.Errorf("error verifying access token: %+v", err)
			return ctx.Redirect("/login", http.StatusMovedPermanently)
		}

		log.Debugf("user has roles: %v accociated", user.GetRoles())
		ctx.Locals(oidcUserContextKey, user)
	}
	return ctx.Next()
}

// handleAimMlflowResourceRequest applies OIDC check for Aim or Mlflow resources.
func (m OIDCMiddleware) handleAimMlflowResourceRequest(ctx *fiber.Ctx) error {
	namespace, err := GetNamespaceFromContext(ctx.Context())
	if err != nil {
		return api.NewInternalError("error getting namespace from context")
	}
	log.Debugf("checking access permission to %s namespace", namespace.Code)

	authToken := strings.Replace(ctx.Get("Authorization"), "Bearer ", "", 1)
	if authToken == "" {
		log.Error("auth token has incorrect format")
		return ctx.Status(
			http.StatusNotFound,
		).JSON(
			api.NewResourceDoesNotExistError("unable to find namespace with code: %s", namespace.Code),
		)
	}

	user, err := m.client.Verify(ctx.Context(), authToken)
	if err != nil {
		return ctx.Status(
			http.StatusNotFound,
		).JSON(
			api.NewResourceDoesNotExistError("unable to find namespace with code: %s", namespace.Code),
		)
	}
	log.Debugf("user has roles: %v accociated", user.GetRoles())

	if user.IsAdmin() {
		return ctx.Next()
	}

	isValid, err := m.rolesRepository.ValidateRolesAccessToNamespace(ctx.Context(), user.GetRoles(), namespace.Code)
	if err != nil {
		log.Errorf("error validating access to requested namespace with code: %s, %+v", namespace.Code, err)
		return api.NewInternalError(
			"error validating access to requested namespace with code: %s", namespace.Code,
		)
	}
	if !isValid {
		return ctx.Status(
			http.StatusNotFound,
		).JSON(
			api.NewResourceDoesNotExistError("unable to find namespace with code: %s", namespace.Code),
		)
	}
	return ctx.Next()
}

// GetOIDCUserFromContext returns OIDC User object from the context.
func GetOIDCUserFromContext(ctx context.Context) (*auth.User, error) {
	user, ok := ctx.Value(oidcUserContextKey).(*auth.User)
	if !ok {
		return nil, eris.New("error getting oidc user object from context")
	}
	return user, nil
}
