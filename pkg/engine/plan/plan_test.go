package plan

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/jensneuse/diffview"

	"github.com/jensneuse/graphql-go-tools/internal/pkg/unsafeparser"
	"github.com/jensneuse/graphql-go-tools/pkg/astnormalization"
	"github.com/jensneuse/graphql-go-tools/pkg/asttransform"
	"github.com/jensneuse/graphql-go-tools/pkg/astvalidation"
	"github.com/jensneuse/graphql-go-tools/pkg/engine/resolve"
	"github.com/jensneuse/graphql-go-tools/pkg/operationreport"
)

func TestPlanner_Plan(t *testing.T) {
	test := func(definition, operation, operationName string, expectedPlan Plan) func(t *testing.T) {
		return func(t *testing.T) {
			def := unsafeparser.ParseGraphqlDocumentString(definition)
			op := unsafeparser.ParseGraphqlDocumentString(operation)
			err := asttransform.MergeDefinitionWithBaseSchema(&def)
			if err != nil {
				t.Fatal(err)
			}
			norm := astnormalization.NewNormalizer(true)
			var report operationreport.Report
			norm.NormalizeOperation(&op, &def, &report)
			valid := astvalidation.DefaultOperationValidator()
			valid.Validate(&op, &def, &report)
			p := NewPlanner(&def,Configuration{})
			plan := p.Plan(&op, []byte(operationName), &report)
			if report.HasErrors() {
				t.Fatal(report.Error())
			}
			if !reflect.DeepEqual(expectedPlan, plan) {
				diffview.NewGoland().DiffViewAny("diff", expectedPlan, plan)
				t.Errorf("want:\n%s\ngot:\n%s\n", spew.Sdump(expectedPlan), spew.Sdump(plan))
			}
		}
	}

	t.Run("simple named Query", test(testDefinition, `
		query MyQuery($id: ID!){
			droid(id: $id){
				name
				aliased: name
				friends {
					name
				}
				primaryFunction
			}
		}
	`, "MyQuery", &SynchronousResponsePlan{
		Response: resolve.GraphQLResponse{
			Data: &resolve.Object{
				FieldSets: []resolve.FieldSet{
					{
						Fields: []resolve.Field{
							{
								Name: []byte("droid"),
								Value: &resolve.Object{
									FieldSets: []resolve.FieldSet{
										{
											Fields: []resolve.Field{
												{
													Name: []byte("name"),
													Value: &resolve.String{
														Path: []string{"name"},
													},
												},
												{
													Name: []byte("aliased"),
													Value: &resolve.String{
														Path: []string{"name"},
													},
												},
												{
													Name: []byte("friends"),
													Value: &resolve.Array{
														Path: []string{"friends"},
														Item: &resolve.Object{
															FieldSets: []resolve.FieldSet{
																{
																	Fields: []resolve.Field{
																		{
																			Name: []byte("name"),
																			Value: &resolve.String{
																				Path: []string{"name"},
																			},
																		},
																	},
																},
															},
														},
													},
												},
												{
													Name: []byte("primaryFunction"),
													Value: &resolve.String{
														Path: []string{"primaryFunction"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}))
	t.Run("named Query in Operation with multiple queries", test(testDefinition, `
		query Query1($id: ID!){
			droid(id: $id){
				name
			}
		}
		query Query2($id: ID!){
			droid(id: $id){
				name
				primaryFunction
			}
		}
	`, "Query1", &SynchronousResponsePlan{
		Response: resolve.GraphQLResponse{
			Data: &resolve.Object{
				FieldSets: []resolve.FieldSet{
					{
						Fields: []resolve.Field{
							{
								Name: []byte("droid"),
								Value: &resolve.Object{
									FieldSets: []resolve.FieldSet{
										{
											Fields: []resolve.Field{
												{
													Name: []byte("name"),
													Value: &resolve.String{
														Path: []string{"name"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}))
}

const testDefinition = `
union SearchResult = Human | Droid | Starship

schema {
    query: Query
    mutation: Mutation
    subscription: Subscription
}

type Query {
    hero: Character
    droid(id: ID!): Droid
    search(name: String!): SearchResult
}

type Mutation {
    createReview(episode: Episode!, review: ReviewInput!): Review
}

type Subscription {
    remainingJedis: Int!
}

input ReviewInput {
    stars: Int!
    commentary: String
}

type Review {
    id: ID!
    stars: Int!
    commentary: String
}

enum Episode {
    NEWHOPE
    EMPIRE
    JEDI
}

interface Character {
    name: String!
    friends: [Character]
}

type Human implements Character {
    name: String!
    height: String!
    friends: [Character]
}

type Droid implements Character {
    name: String!
    primaryFunction: String!
    friends: [Character]
}

type Startship {
    name: String!
    length: Float!
}`
