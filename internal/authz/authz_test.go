package authz

import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"

	"github.com/artifacthub/hub/internal/hub"
	"github.com/artifacthub/hub/internal/tests"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	user1ID    = "0001"
	user1Alias = "user1"
	user2ID    = "0002"
	user2Alias = "user2"
	user3ID    = "0003"
	user3Alias = "user3"
	user4ID    = "0004"
	user4Alias = "user4"
	user5ID    = "0005"
	org1Name   = "org1"
	org2Name   = "org2"
	org3Name   = "org3"
)

var testsAuthorizationPoliciesJSON = []byte(`{
	"org1": {
		"authorization_enabled": true,
		"predefined_policy": "rbac.v1",
		"policy_data": {
			"roles": {
				"owner": {
					"users": [
						"user1"
					]
				},
				"admin": {
					"users": [
						"user2"
					],
					"allowed_actions": [
						"addOrganizationMember",
						"deleteOrganizationMember"
					]
				},
				"member": {
					"users": [
						"user3"
					],
					"allowed_actions": [
						"updateOrganization"
					]
				}
			}
		}
	},
	"org2": {
		"authorization_enabled": true,
		"custom_policy": "package artifacthub.authz\ndefault allow = false\n",
		"policy_data": {}
	},
	"org3": {
		"authorization_enabled": false,
		"custom_policy": "package artifacthub.authz\ndefault allow = false\n",
		"policy_data": {}
	}
}`)

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.Disabled)
	os.Exit(m.Run())
}

func TestNewAuthorizer(t *testing.T) {
	dbQuery := "select get_authorization_policies()"

	t.Run("error getting authorization policies", func(t *testing.T) {
		db := &tests.DBMock{}
		db.On("QueryRow", context.Background(), dbQuery).Return(nil, tests.ErrFakeDatabaseFailure)
		db.On("Acquire", context.Background()).Return(nil, tests.ErrFakeDatabaseFailure).Maybe()
		_, err := NewAuthorizer(db)
		assert.True(t, errors.Is(err, tests.ErrFakeDatabaseFailure))
		db.AssertExpectations(t)
	})

	t.Run("error unmarshalling authorization policies", func(t *testing.T) {
		db := &tests.DBMock{}
		db.On("QueryRow", context.Background(), dbQuery).Return([]byte(`{"invalid`), nil)
		db.On("Acquire", context.Background()).Return(nil, tests.ErrFakeDatabaseFailure).Maybe()
		_, err := NewAuthorizer(db)
		assert.Error(t, err)
		db.AssertExpectations(t)
	})

	t.Run("authorizer created successfully", func(t *testing.T) {
		db := &tests.DBMock{}
		db.On("QueryRow", context.Background(), dbQuery).Return(testsAuthorizationPoliciesJSON, nil)
		db.On("Acquire", context.Background()).Return(nil, tests.ErrFakeDatabaseFailure).Maybe()
		az, err := NewAuthorizer(db)
		assert.Contains(t, az.allowQueries, "org1")
		assert.Contains(t, az.allowedActionsQueries, "org1")
		assert.Nil(t, err)
		db.AssertExpectations(t)
	})
}

func TestAuthorize(t *testing.T) {
	policiesQuery := "select get_authorization_policies()"
	userAliasQuery := `select alias from "user" where user_id = $1`
	db := &tests.DBMock{}
	db.On("QueryRow", context.Background(), policiesQuery).Return(testsAuthorizationPoliciesJSON, nil)
	db.On("QueryRow", context.Background(), userAliasQuery, user1ID).Return(user1Alias, nil).Maybe()
	db.On("QueryRow", context.Background(), userAliasQuery, user2ID).Return(user2Alias, nil).Maybe()
	db.On("QueryRow", context.Background(), userAliasQuery, user3ID).Return(user3Alias, nil).Maybe()
	db.On("QueryRow", context.Background(), userAliasQuery, user5ID).Return("", tests.ErrFakeDatabaseFailure).Maybe()
	db.On("Acquire", context.Background()).Return(nil, tests.ErrFakeDatabaseFailure).Maybe()
	az, err := NewAuthorizer(db)
	require.NoError(t, err)

	testCases := []struct {
		input *hub.AuthorizeInput
		allow bool
	}{
		{
			&hub.AuthorizeInput{
				OrganizationName: org1Name,
				UserID:           user1ID,
				Action:           hub.AddOrganizationMember,
			},
			true,
		},
		{
			&hub.AuthorizeInput{
				OrganizationName: org1Name,
				UserID:           user2ID,
				Action:           hub.AddOrganizationMember,
			},
			true,
		},
		{
			&hub.AuthorizeInput{
				OrganizationName: org1Name,
				UserID:           user3ID,
				Action:           hub.AddOrganizationMember,
			},
			false,
		},
		{
			&hub.AuthorizeInput{
				OrganizationName: org1Name,
				UserID:           user5ID,
				Action:           hub.AddOrganizationMember,
			},
			false,
		},
		{
			&hub.AuthorizeInput{
				OrganizationName: org1Name,
				UserID:           user2ID,
				Action:           hub.UpdateOrganization,
			},
			false,
		},
		{
			&hub.AuthorizeInput{
				OrganizationName: org1Name,
				UserID:           user3ID,
				Action:           hub.UpdateOrganization,
			},
			true,
		},
		{
			&hub.AuthorizeInput{
				OrganizationName: org1Name,
				UserID:           user1ID,
				Action:           hub.TransferOrganizationRepository,
			},
			true,
		},
		{
			&hub.AuthorizeInput{
				OrganizationName: org1Name,
				UserID:           user2ID,
				Action:           hub.TransferOrganizationRepository,
			},
			false,
		},
		{
			&hub.AuthorizeInput{
				OrganizationName: org2Name,
				UserID:           user1ID,
				Action:           hub.AddOrganizationMember,
			},
			false,
		},
		{
			&hub.AuthorizeInput{
				OrganizationName: org2Name,
				UserID:           user2ID,
				Action:           hub.AddOrganizationMember,
			},
			false,
		},
		{
			&hub.AuthorizeInput{
				OrganizationName: org3Name,
				UserID:           user1ID,
				Action:           hub.AddOrganizationMember,
			},
			true,
		},
		{
			&hub.AuthorizeInput{
				OrganizationName: org3Name,
				UserID:           user2ID,
				Action:           hub.AddOrganizationMember,
			},
			true,
		},
		{
			&hub.AuthorizeInput{
				OrganizationName: org3Name,
				UserID:           user3ID,
				Action:           hub.AddOrganizationMember,
			},
			true,
		},
	}
	for i, tc := range testCases {
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			err := az.Authorize(context.Background(), tc.input)
			if tc.allow {
				assert.Nil(t, err)
			} else {
				assert.True(t, errors.Is(err, hub.ErrInsufficientPrivilege))
			}
		})
	}

	db.AssertExpectations(t)
}

func TestGetAllowedActions(t *testing.T) {
	policiesQuery := "select get_authorization_policies()"
	userAliasQuery := `select alias from "user" where user_id = $1`
	db := &tests.DBMock{}
	db.On("QueryRow", context.Background(), policiesQuery).Return(testsAuthorizationPoliciesJSON, nil)
	db.On("QueryRow", context.Background(), userAliasQuery, user1ID).Return(user1Alias, nil).Maybe()
	db.On("QueryRow", context.Background(), userAliasQuery, user2ID).Return(user2Alias, nil).Maybe()
	db.On("QueryRow", context.Background(), userAliasQuery, user3ID).Return(user3Alias, nil).Maybe()
	db.On("QueryRow", context.Background(), userAliasQuery, user4ID).Return(user4Alias, nil).Maybe()
	db.On("QueryRow", context.Background(), userAliasQuery, user5ID).Return("", tests.ErrFakeDatabaseFailure).Maybe()
	db.On("Acquire", context.Background()).Return(nil, tests.ErrFakeDatabaseFailure).Maybe()
	az, err := NewAuthorizer(db)
	require.NoError(t, err)

	testCases := []struct {
		userID                 string
		orgName                string
		expectedAllowedActions []hub.Action
	}{
		{
			user1ID,
			org1Name,
			[]hub.Action{
				hub.Action("all"),
			},
		},
		{
			user2ID,
			org1Name,
			[]hub.Action{
				hub.AddOrganizationMember,
				hub.DeleteOrganizationMember,
			},
		},
		{
			user3ID,
			org1Name,
			[]hub.Action{
				hub.UpdateOrganization,
			},
		},
		{
			user4ID,
			org1Name,
			[]hub.Action{},
		},
		{
			user5ID,
			org1Name,
			nil,
		},
		{
			user1ID,
			org2Name,
			nil,
		},
		{
			user2ID,
			org2Name,
			nil,
		},
		{
			user1ID,
			org3Name,
			[]hub.Action{
				hub.Action("all"),
			},
		},
		{
			user3ID,
			org3Name,
			[]hub.Action{
				hub.Action("all"),
			},
		},
	}
	for i, tc := range testCases {
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			allowedActions, _ := az.GetAllowedActions(context.Background(), tc.userID, tc.orgName)
			assert.Equal(t, tc.expectedAllowedActions, allowedActions)
		})
	}

	db.AssertExpectations(t)
}