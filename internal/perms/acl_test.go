package perms

import (
	"testing"

	"github.com/hashicorp/watchtower/internal/types/action"
	"github.com/hashicorp/watchtower/internal/types/resource"
	"github.com/hashicorp/watchtower/internal/types/scope"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ACLAllowed(t *testing.T) {
	t.Parallel()

	type scopeGrant struct {
		scope  Scope
		grants []string
	}
	type actionAllowed struct {
		action  action.Type
		allowed bool
	}
	type input struct {
		name           string
		scopeGrants    []scopeGrant
		resource       Resource
		actionsAllowed []actionAllowed
		userId         string
	}

	// A set of common grants to use in the following tests
	commonGrants := []scopeGrant{
		{
			scope: Scope{Type: scope.Organization, Id: "org_a"},
			grants: []string{
				"id=a_bar;actions=read,update",
				"id=a_foo;type=host-catalog;actions=update,delete",
				"type=host-set;actions=list,create",
			},
		},
		{
			scope: Scope{Type: scope.Organization, Id: "org_b"},
			grants: []string{
				"type=host-set;actions=list,create",
				"type=host;actions=*",
				"id=*;actions=authenticate",
			},
		},
		{
			scope: Scope{Type: scope.Organization, Id: "org_c"},
			grants: []string{
				"id={{user.id }};actions=create,update",
			},
		},
	}

	// See acl.go for expected allowed formats. The goal here is to basically
	// test those, but also test a whole bunch of nonconforming values.
	tests := []input{
		{
			name:     "no grants",
			resource: Resource{ScopeId: "foo", Id: "bar", Type: resource.HostCatalog},
			actionsAllowed: []actionAllowed{
				{action: action.Create},
				{action: action.Read},
			},
		},
		{
			name:        "no overlap",
			resource:    Resource{ScopeId: "foo", Id: "bar", Type: resource.HostCatalog},
			scopeGrants: commonGrants,
			actionsAllowed: []actionAllowed{
				{action: action.Create},
				{action: action.Read},
			},
		},
		{
			name:        "matching scope and id no matching action",
			resource:    Resource{ScopeId: "org_a", Id: "a_foo"},
			scopeGrants: commonGrants,
			actionsAllowed: []actionAllowed{
				{action: action.Update},
				{action: action.Delete},
			},
		},
		{
			name:        "matching scope and id and matching action",
			resource:    Resource{ScopeId: "org_a", Id: "a_bar"},
			scopeGrants: commonGrants,
			actionsAllowed: []actionAllowed{
				{action: action.Read, allowed: true},
				{action: action.Update, allowed: true},
				{action: action.Delete},
			},
		},
		{
			name:        "matching scope and id and all action",
			resource:    Resource{ScopeId: "org_b", Type: resource.Host},
			scopeGrants: commonGrants,
			actionsAllowed: []actionAllowed{
				{action: action.Read, allowed: true},
				{action: action.Update, allowed: true},
				{action: action.Delete, allowed: true},
			},
		},
		{
			name:        "matching scope and id and all action but bad specifier",
			resource:    Resource{ScopeId: "org_b", Id: "id_g"},
			scopeGrants: commonGrants,
			actionsAllowed: []actionAllowed{
				{action: action.Read},
				{action: action.Update},
				{action: action.Delete},
			},
		},
		{
			name:        "matching scope and not matching type",
			resource:    Resource{ScopeId: "org_a", Type: resource.HostCatalog},
			scopeGrants: commonGrants,
			actionsAllowed: []actionAllowed{
				{action: action.Update},
				{action: action.Delete},
			},
		},
		{
			name:        "matching scope and matching type",
			resource:    Resource{ScopeId: "org_a", Type: resource.HostSet},
			scopeGrants: commonGrants,
			actionsAllowed: []actionAllowed{
				{action: action.List, allowed: true},
				{action: action.Create, allowed: true},
				{action: action.Delete},
			},
		},
		{
			name:        "matching scope, type, action, bad pin",
			resource:    Resource{ScopeId: "org_a", Id: "a_foo", Type: resource.HostCatalog},
			scopeGrants: commonGrants,
			actionsAllowed: []actionAllowed{
				{action: action.Update},
				{action: action.Delete},
				{action: action.Read},
			},
		},
		{
			name:        "matching scope, type, action, random id and bad pin",
			resource:    Resource{ScopeId: "org_a", Id: "anything", Type: resource.HostCatalog, Pin: "a_bar"},
			scopeGrants: commonGrants,
			actionsAllowed: []actionAllowed{
				{action: action.Update},
				{action: action.Delete},
				{action: action.Read},
			},
		},
		{
			name:        "matching scope, type, action, random id and good pin",
			resource:    Resource{ScopeId: "org_a", Id: "anything", Type: resource.HostCatalog, Pin: "a_foo"},
			scopeGrants: commonGrants,
			actionsAllowed: []actionAllowed{
				{action: action.Update, allowed: true},
				{action: action.Delete, allowed: true},
				{action: action.Read},
			},
		},
		{
			name:        "wrong scope and matching type",
			resource:    Resource{ScopeId: "org_bad", Type: resource.HostSet},
			scopeGrants: commonGrants,
			actionsAllowed: []actionAllowed{
				{action: action.List},
				{action: action.Create},
				{action: action.Delete},
			},
		},
		{
			name:        "any id",
			resource:    Resource{ScopeId: "org_b", Type: resource.AuthMethod},
			scopeGrants: commonGrants,
			actionsAllowed: []actionAllowed{
				{action: action.List},
				{action: action.Authenticate, allowed: true},
				{action: action.Delete},
			},
		},
		{
			name:        "bad templated user id",
			resource:    Resource{ScopeId: "org_c"},
			scopeGrants: commonGrants,
			actionsAllowed: []actionAllowed{
				{action: action.List},
				{action: action.Authenticate},
				{action: action.Delete},
			},
			userId: "u_abcd1234",
		},
		{
			name:        "good templated user id",
			resource:    Resource{ScopeId: "org_c", Id: "u_abcd1234"},
			scopeGrants: commonGrants,
			actionsAllowed: []actionAllowed{
				{action: action.Create, allowed: true},
				{action: action.Update, allowed: true},
			},
			userId: "u_abcd1234",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var grants []Grant
			for _, sg := range test.scopeGrants {
				scope := sg.scope
				for _, g := range sg.grants {
					grant, err := Parse(scope, test.userId, g)
					require.NoError(t, err)
					grants = append(grants, grant)
				}
			}
			acl := NewACL(grants...)
			for _, aa := range test.actionsAllowed {
				assert.True(t, acl.Allowed(test.resource, aa.action).Allowed == aa.allowed)
			}
		})
	}
}
