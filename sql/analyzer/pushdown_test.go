package analyzer

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dolthub/go-mysql-server/memory"
	"github.com/dolthub/go-mysql-server/sql"
	"github.com/dolthub/go-mysql-server/sql/expression"
	"github.com/dolthub/go-mysql-server/sql/plan"
)

func TestPushdownProjectionToTables(t *testing.T) {
	table := memory.NewPushdownTable("mytable", sql.Schema{
		{Name: "i", Type: sql.Int32, Source: "mytable"},
		{Name: "f", Type: sql.Float64, Source: "mytable"},
		{Name: "t", Type: sql.Text, Source: "mytable"},
	})

	table2 := memory.NewPushdownTable("mytable2", sql.Schema{
		{Name: "i2", Type: sql.Int32, Source: "mytable2"},
		{Name: "f2", Type: sql.Float64, Source: "mytable2"},
		{Name: "t2", Type: sql.Text, Source: "mytable2"},
	})

	db := memory.NewDatabase("mydb")
	db.AddTable("mytable", table)
	db.AddTable("mytable2", table2)

	catalog := sql.NewCatalog()
	catalog.AddDatabase(db)
	a := NewDefault(catalog)

	tests := []analyzerFnTestCase{
		{
			name: "pushdown projections to tables",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(2, sql.Text, "mytable2", "t2", false),
				},
				plan.NewFilter(
					expression.NewOr(
						expression.NewEquals(
							expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", false),
							expression.NewLiteral(3.14, sql.Float64),
						),
						expression.NewIsNull(
							expression.NewGetFieldWithTable(0, sql.Int32, "mytable2", "i2", false),
						),
					),
					plan.NewCrossJoin(
						plan.NewResolvedTable(table),
						plan.NewResolvedTable(table2),
					),
				),
			),
			expected: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(1, sql.Text, "mytable2", "t2", false),
				},
				plan.NewFilter(
					expression.NewOr(
						expression.NewEquals(
							expression.NewGetFieldWithTable(0, sql.Float64, "mytable", "f", false),
							expression.NewLiteral(3.14, sql.Float64),
						),
						expression.NewIsNull(
							expression.NewGetFieldWithTable(2, sql.Int32, "mytable2", "i2", false),
						),
					),
					plan.NewCrossJoin(
						plan.NewDecoratedNode(plan.DecorationTypeProjectedAccess, "Projected table access on [f]", plan.NewResolvedTable(
							table.WithProjection([]string{"f"}),
						)),
						plan.NewDecoratedNode(plan.DecorationTypeProjectedAccess, "Projected table access on [t2 i2]", plan.NewResolvedTable(
							table2.WithProjection([]string{"t2", "i2"}),
						)),
					),
				),
			),
		},
		{
			name: "pushing projections down onto a filtered table",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(5, sql.Text, "mytable2", "t2", false),
				},
				plan.NewCrossJoin(
					plan.NewDecoratedNode(plan.DecorationTypeFilteredAccess, "Filtered table access on [mytable.f = 3.14]", plan.NewResolvedTable(
						table.WithFilters([]sql.Expression{
							expression.NewEquals(
								expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", false),
								expression.NewLiteral(3.14, sql.Float64),
							),
						}),
					)),
					plan.NewDecoratedNode(plan.DecorationTypeFilteredAccess, "Filtered table access on [mytable2.i2 IS NULL]", plan.NewResolvedTable(
						table2.WithFilters([]sql.Expression{
							expression.NewIsNull(
								expression.NewGetFieldWithTable(0, sql.Int32, "mytable2", "i2", false),
							),
						}),
					)),
				),
			),
			expected: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(3, sql.Text, "mytable2", "t2", false),
				},
				plan.NewCrossJoin(
					plan.NewDecoratedNode(plan.DecorationTypeFilteredAccess, "Filtered table access on [mytable.f = 3.14]", plan.NewResolvedTable(
						table.WithFilters([]sql.Expression{
							expression.NewEquals(
								expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", false),
								expression.NewLiteral(3.14, sql.Float64),
							),
						}),
					)),
					plan.NewDecoratedNode(plan.DecorationTypeFilteredAccess, "Filtered table access on [mytable2.i2 IS NULL]",
						plan.NewDecoratedNode(plan.DecorationTypeProjectedAccess, "Projected table access on [t2]",
							plan.NewResolvedTable(
								table2.WithFilters([]sql.Expression{
									expression.NewIsNull(
										expression.NewGetFieldWithTable(0, sql.Int32, "mytable2", "i2", false),
									),
								}).(*memory.PushdownTable).WithProjection([]string{"t2"}),
							),
						),
					),
				),
			),
		},
	}

	runTestCases(t, sql.NewEmptyContext(), tests, a, getRule("pushdown_projections"))
}

func TestPushdownFilterToTables(t *testing.T) {
	table := memory.NewPushdownTable("mytable", sql.Schema{
		{Name: "i", Type: sql.Int32, Source: "mytable"},
		{Name: "f", Type: sql.Float64, Source: "mytable"},
		{Name: "t", Type: sql.Text, Source: "mytable"},
	})

	table2 := memory.NewPushdownTable("mytable2", sql.Schema{
		{Name: "i2", Type: sql.Int32, Source: "mytable2"},
		{Name: "f2", Type: sql.Float64, Source: "mytable2"},
		{Name: "t2", Type: sql.Text, Source: "mytable2"},
	})

	db := memory.NewDatabase("mydb")
	db.AddTable("mytable", table)
	db.AddTable("mytable2", table2)

	catalog := sql.NewCatalog()
	catalog.AddDatabase(db)
	a := NewDefault(catalog)

	tests := []analyzerFnTestCase{
		{
			name: "pushdown filter to tables",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(2, sql.Text, "mytable2", "t2", false),
				},
				plan.NewFilter(
					expression.NewAnd(
						expression.NewEquals(
							expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", false),
							expression.NewLiteral(3.14, sql.Float64),
						),
						expression.NewIsNull(
							expression.NewGetFieldWithTable(0, sql.Int32, "mytable2", "i2", false),
						),
					),
					plan.NewCrossJoin(
						plan.NewResolvedTable(table),
						plan.NewResolvedTable(table2),
					),
				),
			),
			expected: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(5, sql.Text, "mytable2", "t2", false),
				},
				plan.NewCrossJoin(
					plan.NewDecoratedNode(plan.DecorationTypeFilteredAccess, "Filtered table access on [mytable.f = 3.14]", plan.NewResolvedTable(
						table.WithFilters([]sql.Expression{
							expression.NewEquals(
								expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", false),
								expression.NewLiteral(3.14, sql.Float64),
							),
						}),
					)),
					plan.NewDecoratedNode(plan.DecorationTypeFilteredAccess, "Filtered table access on [mytable2.i2 IS NULL]", plan.NewResolvedTable(
						table2.WithFilters([]sql.Expression{
							expression.NewIsNull(
								expression.NewGetFieldWithTable(0, sql.Int32, "mytable2", "i2", false),
							),
						}),
					)),
				),
			),
		},
		{
			name: "push filters down onto projected table",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(1, sql.Text, "mytable2", "t2", false),
				},
				plan.NewFilter(
					expression.NewAnd(
						expression.NewEquals(
							expression.NewGetFieldWithTable(0, sql.Float64, "mytable", "f", false),
							expression.NewLiteral(3.14, sql.Float64),
						),
						expression.NewIsNull(
							expression.NewGetFieldWithTable(2, sql.Int32, "mytable2", "i2", false),
						),
					),
					plan.NewCrossJoin(
						plan.NewDecoratedNode(plan.DecorationTypeProjectedAccess, "Projected table access on [f]",
							plan.NewResolvedTable(
								table.WithProjection([]string{"f"}),
							),
						),
						plan.NewDecoratedNode(plan.DecorationTypeProjectedAccess, "Projected table access on [t2 i2]",
							plan.NewResolvedTable(
								table2.WithProjection([]string{"t2", "i2"}),
							),
						),
					),
				),
			),
			expected: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(1, sql.Text, "mytable2", "t2", false),
				},
				plan.NewCrossJoin(
					plan.NewDecoratedNode(plan.DecorationTypeProjectedAccess, "Projected table access on [f]",
						plan.NewDecoratedNode(plan.DecorationTypeFilteredAccess, "Filtered table access on [mytable.f = 3.14]",
							plan.NewResolvedTable(
								table.WithProjection([]string{"f"}).(*memory.PushdownTable).WithFilters([]sql.Expression{
									eq(expression.NewGetFieldWithTable(0, sql.Float64, "mytable", "f", false), expression.NewLiteral(3.14, sql.Float64)),
								}),
							),
						),
					),
					plan.NewDecoratedNode(plan.DecorationTypeProjectedAccess, "Projected table access on [t2 i2]",
						plan.NewDecoratedNode(plan.DecorationTypeFilteredAccess, "Filtered table access on [mytable2.i2 IS NULL]",
							plan.NewResolvedTable(
								table2.WithProjection([]string{"t2", "i2"}).(*memory.PushdownTable).WithFilters([]sql.Expression{
									expression.NewIsNull(expression.NewGetFieldWithTable(1, sql.Int32, "mytable2", "i2", false)),
								}),
							),
						),
					),
				),
			),
		},
	}

	runTestCases(t, sql.NewEmptyContext(), tests, a, getRule("pushdown_filters"))
}

func TestPushdownFiltersAboveTables(t *testing.T) {
	table := memory.NewTable("mytable", sql.Schema{
		{Name: "i", Type: sql.Int32, Source: "mytable"},
		{Name: "f", Type: sql.Float64, Source: "mytable"},
		{Name: "t", Type: sql.Text, Source: "mytable"},
	})

	table2 := memory.NewTable("mytable2", sql.Schema{
		{Name: "i2", Type: sql.Int32, Source: "mytable2"},
		{Name: "f2", Type: sql.Float64, Source: "mytable2"},
		{Name: "t2", Type: sql.Text, Source: "mytable2"},
	})

	db := memory.NewDatabase("mydb")
	db.AddTable("mytable", table)
	db.AddTable("mytable2", table2)

	catalog := sql.NewCatalog()
	catalog.AddDatabase(db)
	a := NewDefault(catalog)

	tests := []analyzerFnTestCase{
		{
			name: "pushdown filters to under join node",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", true),
				},
				plan.NewFilter(
					and(
						expression.NewEquals(
							expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", true),
							expression.NewLiteral(3.14, sql.Float64),
						),
						and(
							expression.NewEquals(
								expression.NewGetFieldWithTable(3, sql.Int32, "mytable2", "i2", true),
								expression.NewLiteral(21, sql.Int32),
							),
							expression.NewEquals(
								expression.NewGetFieldWithTable(5, sql.Int32, "mytable2", "t2", true),
								expression.NewLiteral("hello", sql.Text),
							),
						),
					),
					plan.NewCrossJoin(
						plan.NewResolvedTable(table),
						plan.NewResolvedTable(table2),
					),
				),
			),
			expected: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", true),
				},
				plan.NewCrossJoin(
					plan.NewFilter(
						expression.NewEquals(
							expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", true),
							expression.NewLiteral(3.14, sql.Float64),
						),
						plan.NewResolvedTable(table),
					),
					plan.NewFilter(
						and(
							expression.NewEquals(
								expression.NewGetFieldWithTable(0, sql.Int32, "mytable2", "i2", true),
								expression.NewLiteral(21, sql.Int32),
							),
							expression.NewEquals(
								expression.NewGetFieldWithTable(2, sql.Int32, "mytable2", "t2", true),
								expression.NewLiteral("hello", sql.Text),
							),
						),
						plan.NewResolvedTable(table2),
					),
				),
			),
		},
		{
			name: "pushdown filters to under join node, aliased tables",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "t1", "i", true),
				},
				plan.NewFilter(
					and(
						expression.NewEquals(
							expression.NewGetFieldWithTable(1, sql.Float64, "t1", "f", true),
							expression.NewLiteral(3.14, sql.Float64),
						),
						and(
							expression.NewEquals(
								expression.NewGetFieldWithTable(3, sql.Int32, "t2", "i2", true),
								expression.NewLiteral(21, sql.Int32),
							),
							expression.NewEquals(
								expression.NewGetFieldWithTable(5, sql.Int32, "t2", "t2", true),
								expression.NewLiteral("hello", sql.Text),
							),
						),
					),
					plan.NewCrossJoin(
						plan.NewTableAlias("t1",
							plan.NewResolvedTable(table),
						),
						plan.NewTableAlias("t2",
							plan.NewResolvedTable(table2),
						),
					),
				),
			),
			expected: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "t1", "i", true),
				},
				plan.NewCrossJoin(
					plan.NewFilter(
						expression.NewEquals(
							expression.NewGetFieldWithTable(1, sql.Float64, "t1", "f", true),
							expression.NewLiteral(3.14, sql.Float64),
						),
						plan.NewTableAlias("t1",
							plan.NewResolvedTable(table),
						),
					),
					plan.NewFilter(
						and(
							expression.NewEquals(
								expression.NewGetFieldWithTable(0, sql.Int32, "t2", "i2", true),
								expression.NewLiteral(21, sql.Int32),
							),
							expression.NewEquals(
								expression.NewGetFieldWithTable(2, sql.Int32, "t2", "t2", true),
								expression.NewLiteral("hello", sql.Text),
							),
						),
						plan.NewTableAlias("t2",
							plan.NewResolvedTable(table2),
						),
					),
				),
			),
		},
		{
			name: "pushdown filter to left join",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(2, sql.Text, "mytable2", "t2", false),
				},
				plan.NewFilter(
					expression.NewAnd(
						expression.NewEquals(
							expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", false),
							expression.NewLiteral(3.14, sql.Float64),
						),
						expression.NewIsNull(
							expression.NewGetFieldWithTable(0, sql.Int32, "mytable2", "i2", false),
						),
					),
					plan.NewLeftJoin(
						plan.NewResolvedTable(table),
						plan.NewResolvedTable(table2),
						eq(gf(0, "mytable", "i"), gf(3, "mytable2", "i2")),
					),
				),
			),
			expected: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(5, sql.Text, "mytable2", "t2", false),
				},
				plan.NewFilter(
					expression.NewIsNull(
						expression.NewGetFieldWithTable(3, sql.Int32, "mytable2", "i2", false),
					),
					plan.NewLeftJoin(
						plan.NewFilter(
							expression.NewEquals(
								expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", false),
								expression.NewLiteral(3.14, sql.Float64),
							),
							plan.NewResolvedTable(table),
						),
						plan.NewResolvedTable(table2),
						eq(gf(0, "mytable", "i"), gf(3, "mytable2", "i2")),
					),
				),
			),
		},
		{
			name: "pushdown filter to right join",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(2, sql.Text, "mytable2", "t2", false),
				},
				plan.NewFilter(
					expression.NewAnd(
						expression.NewEquals(
							expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", false),
							expression.NewLiteral(3.14, sql.Float64),
						),
						expression.NewIsNull(
							expression.NewGetFieldWithTable(0, sql.Int32, "mytable2", "i2", false),
						),
					),
					plan.NewRightJoin(
						plan.NewResolvedTable(table),
						plan.NewResolvedTable(table2),
						eq(gf(0, "mytable", "i"), gf(3, "mytable2", "i2")),
					),
				),
			),
			expected: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(5, sql.Text, "mytable2", "t2", false),
				},
				plan.NewFilter(
					expression.NewEquals(
						expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", false),
						expression.NewLiteral(3.14, sql.Float64),
					),
					plan.NewRightJoin(
						plan.NewResolvedTable(table),
						plan.NewFilter(
							expression.NewIsNull(
								expression.NewGetFieldWithTable(0, sql.Int32, "mytable2", "i2", false),
							),
							plan.NewResolvedTable(table2),
						),
						eq(gf(0, "mytable", "i"), gf(3, "mytable2", "i2")),
					),
				),
			),
		},
		{
			// TODO: we could push down only the non-join predicates, but we currently just pass entirely
			name: "filter contains join condition (no pushdown currently possible, but see TODO)",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", true),
				},
				plan.NewFilter(
					and(
						expression.NewEquals(
							expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", true),
							expression.NewLiteral(3.14, sql.Float64),
						),
						and(
							expression.NewEquals(
								expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", true),
								expression.NewGetFieldWithTable(3, sql.Int32, "mytable2", "i2", true),
							),
							expression.NewEquals(
								expression.NewGetFieldWithTable(3, sql.Int32, "mytable2", "i2", true),
								expression.NewLiteral(20, sql.Int32),
							),
						),
					),
					plan.NewCrossJoin(
						plan.NewResolvedTable(table),
						plan.NewResolvedTable(table2),
					),
				),
			),
		},
	}

	runTestCases(t, sql.NewEmptyContext(), tests, a, getRule("pushdown_filters"))
}

// TODO: this needs tests for pushing a merged index lookup down to a table
func TestPushdownIndex(t *testing.T) {
	require := require.New(t)

	table := memory.NewTable("mytable", sql.Schema{
		{Name: "i", Type: sql.Int32, Source: "mytable", PrimaryKey: true},
		{Name: "f", Type: sql.Float64, Source: "mytable"},
		{Name: "t", Type: sql.Text, Source: "mytable"},
	})

	table.EnablePrimaryKeyIndexes()
	err := table.CreateIndex(sql.NewEmptyContext(), "f", sql.IndexUsing_BTree, sql.IndexConstraint_None, []sql.IndexColumn{
		{
			Name:   "f",
			Length: 0,
		},
	}, "")
	require.NoError(err)

	table1Idxes, err := table.GetIndexes(sql.NewEmptyContext())
	require.NoError(err)
	idxtable1I := table1Idxes[0]
	fmt.Sprintf("%v", idxtable1I)
	idxTable1F := table1Idxes[1]

	table2 := memory.NewTable("mytable2", sql.Schema{
		{Name: "i2", Type: sql.Int32, Source: "mytable2", PrimaryKey: true},
		{Name: "f2", Type: sql.Float64, Source: "mytable2"},
		{Name: "t2", Type: sql.Text, Source: "mytable2"},
	})

	table2.EnablePrimaryKeyIndexes()
	table2Idxes, err := table2.GetIndexes(sql.NewEmptyContext())
	require.NoError(err)
	idxTable2I2 := table2Idxes[0]

	db := memory.NewDatabase("")
	db.AddTable("mytable", table)
	db.AddTable("mytable2", table2)

	catalog := sql.NewCatalog()
	catalog.AddDatabase(db)
	a := NewDefault(catalog)

	// TODO: the order of operations here means that the decorator node gets separated from the table it decorates
	//  sometimes. Just a cosmetic issue, but should fix.
	tests := []analyzerFnTestCase{
		{
			name: "single index",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", true),
				},
				plan.NewFilter(
					expression.NewEquals(
						expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", true),
						expression.NewLiteral(3.14, sql.Float64),
					),
					plan.NewResolvedTable(table),
				),
			),
			expected: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", true),
				},
				plan.NewDecoratedNode(plan.DecorationTypeIndexedAccess, "Indexed table access on index [mytable.f]",
					plan.NewFilter(
						expression.NewEquals(
							expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", true),
							expression.NewLiteral(3.14, sql.Float64),
						),
						plan.NewResolvedTable(
							table.WithIndexLookup(
								mustIndexLookup(idxTable1F.Get(3.14)),
							),
						),
					),
				),
			),
		},
		{
			name: "single index with extra predicate",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", true),
				},
				plan.NewFilter(
					and(
						expression.NewEquals(
							expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", true),
							expression.NewLiteral(3.14, sql.Float64),
						),
						expression.NewEquals(
							expression.NewGetFieldWithTable(2, sql.Text, "mytable", "t", true),
							expression.NewLiteral("hello", sql.Text),
						),
					),
					plan.NewResolvedTable(table),
				),
			),
			expected: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", true),
				},
				plan.NewDecoratedNode(plan.DecorationTypeIndexedAccess, "Indexed table access on index [mytable.f]",
					plan.NewFilter(
						and(
							expression.NewEquals(
								expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", true),
								expression.NewLiteral(3.14, sql.Float64),
							),
							expression.NewEquals(
								expression.NewGetFieldWithTable(2, sql.Text, "mytable", "t", true),
								expression.NewLiteral("hello", sql.Text),
							),
						),
						plan.NewResolvedTable(
							table.WithIndexLookup(
								mustIndexLookup(idxTable1F.Get(3.14)),
							),
						),
					),
				),
			),
		},
		{
			name: "single index with extra predicates",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", true),
				},
				plan.NewFilter(
					and(
						expression.NewEquals(
							expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", true),
							expression.NewLiteral(3.14, sql.Float64),
						),
						and(
							expression.NewEquals(
								expression.NewGetFieldWithTable(2, sql.Text, "mytable", "t", true),
								expression.NewLiteral("hello", sql.Text),
							),
							expression.NewEquals(
								expression.NewGetFieldWithTable(2, sql.Text, "mytable", "t", true),
								expression.NewLiteral("goodbye", sql.Text),
							),
						),
					),
					plan.NewResolvedTable(table),
				),
			),
			expected: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", true),
				},
				plan.NewDecoratedNode(plan.DecorationTypeIndexedAccess, "Indexed table access on index [mytable.f]",
					plan.NewFilter(
						and(
							and(
								expression.NewEquals(
									expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", true),
									expression.NewLiteral(3.14, sql.Float64),
								),
								expression.NewEquals(
									expression.NewGetFieldWithTable(2, sql.Text, "mytable", "t", true),
									expression.NewLiteral("hello", sql.Text),
								),
							),
							expression.NewEquals(
								expression.NewGetFieldWithTable(2, sql.Text, "mytable", "t", true),
								expression.NewLiteral("goodbye", sql.Text),
							),
						),
						plan.NewResolvedTable(
							table.WithIndexLookup(
								mustIndexLookup(idxTable1F.Get(3.14)),
							),
						),
					),
				),
			),
		},
		{
			name: "single index to each of two tables",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", true),
				},
				plan.NewFilter(
					and(
						expression.NewEquals(
							expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", true),
							expression.NewLiteral(3.14, sql.Float64),
						),
						expression.NewEquals(
							expression.NewGetFieldWithTable(3, sql.Int32, "mytable2", "i2", true),
							expression.NewLiteral(21, sql.Int32),
						),
					),
					plan.NewCrossJoin(
						plan.NewResolvedTable(table),
						plan.NewResolvedTable(table2),
					),
				),
			),
			expected: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", true),
				},
				plan.NewCrossJoin(
					plan.NewDecoratedNode(plan.DecorationTypeIndexedAccess, "Indexed table access on index [mytable.f]",
						plan.NewFilter(
							expression.NewEquals(
								expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", true),
								expression.NewLiteral(3.14, sql.Float64),
							),
							plan.NewResolvedTable(table.WithIndexLookup(
								mustIndexLookup(idxTable1F.Get(3.14))),
							),
						),
					),
					plan.NewDecoratedNode(plan.DecorationTypeIndexedAccess, "Indexed table access on index [mytable2.i2]",
						plan.NewFilter(
							expression.NewEquals(
								expression.NewGetFieldWithTable(0, sql.Int32, "mytable2", "i2", true),
								expression.NewLiteral(21, sql.Int32),
							),
							plan.NewResolvedTable(table2.WithIndexLookup(
								mustIndexLookup(idxTable2I2.Get(21))),
							),
						),
					),
				),
			),
		},
		{
			// This scenario can't happen in the current analyzer rule ordering. But the rule should behave correctly anyway.
			name: "single index to each of two tables, filters already pushed down",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", true),
				},
				plan.NewCrossJoin(
					plan.NewFilter(
						expression.NewEquals(
							expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", true),
							expression.NewLiteral(3.14, sql.Float64),
						),
						plan.NewResolvedTable(table),
					),
					plan.NewFilter(
						expression.NewEquals(
							expression.NewGetFieldWithTable(0, sql.Int32, "mytable2", "i2", true),
							expression.NewLiteral(21, sql.Int32),
						),
						plan.NewResolvedTable(table2),
					),
				),
			),
			expected: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", true),
				},
				plan.NewCrossJoin(
					plan.NewDecoratedNode(plan.DecorationTypeIndexedAccess, "Indexed table access on index [mytable.f]",
						plan.NewFilter(
							expression.NewEquals(
								expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", true),
								expression.NewLiteral(3.14, sql.Float64),
							),
							plan.NewResolvedTable(table.WithIndexLookup(
								mustIndexLookup(idxTable1F.Get(3.14))),
							),
						),
					),
					plan.NewDecoratedNode(plan.DecorationTypeIndexedAccess, "Indexed table access on index [mytable2.i2]",
						plan.NewFilter(
							expression.NewEquals(
								expression.NewGetFieldWithTable(0, sql.Int32, "mytable2", "i2", true),
								expression.NewLiteral(21, sql.Int32),
							),
							plan.NewResolvedTable(table2.WithIndexLookup(
								mustIndexLookup(idxTable2I2.Get(21))),
							),
						),
					),
				),
			),
		},
		{
			name: "Index already pushed down, no change to plan",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", true),
				},
				plan.NewCrossJoin(
					plan.NewDecoratedNode(plan.DecorationTypeIndexedAccess, "Indexed table access on index [mytable.f]",
						plan.NewFilter(
							expression.NewEquals(
								expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", true),
								expression.NewLiteral(3.14, sql.Float64),
							),
							plan.NewResolvedTable(table.WithIndexLookup(
								mustIndexLookup(idxTable1F.Get(3.14))),
							),
						),
					),
					plan.NewDecoratedNode(plan.DecorationTypeIndexedAccess, "Indexed table access on index [mytable2.i2]",
						plan.NewFilter(
							expression.NewEquals(
								expression.NewGetFieldWithTable(0, sql.Int32, "mytable2", "i2", true),
								expression.NewLiteral(21, sql.Int32),
							),
							plan.NewResolvedTable(table2.WithIndexLookup(
								mustIndexLookup(idxTable2I2.Get(21))),
							),
						),
					),
				),
			),
		},
		{
			name: "single index to each of two tables, extra predicates",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", true),
				},
				plan.NewFilter(
					and(
						expression.NewEquals(
							expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", true),
							expression.NewLiteral(3.14, sql.Float64),
						),
						and(
							expression.NewEquals(
								expression.NewGetFieldWithTable(3, sql.Int32, "mytable2", "i2", true),
								expression.NewLiteral(21, sql.Int32),
							),
							expression.NewEquals(
								expression.NewGetFieldWithTable(5, sql.Int32, "mytable2", "t2", true),
								expression.NewLiteral("hello", sql.Text),
							),
						),
					),
					plan.NewCrossJoin(
						plan.NewResolvedTable(table),
						plan.NewResolvedTable(table2),
					),
				),
			),
			expected: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "mytable", "i", true),
				},
				plan.NewCrossJoin(
					plan.NewDecoratedNode(plan.DecorationTypeIndexedAccess, "Indexed table access on index [mytable.f]",
						plan.NewFilter(
							expression.NewEquals(
								expression.NewGetFieldWithTable(1, sql.Float64, "mytable", "f", true),
								expression.NewLiteral(3.14, sql.Float64),
							),
							plan.NewResolvedTable(table.WithIndexLookup(
								mustIndexLookup(idxTable1F.Get(3.14))),
							),
						),
					),
					plan.NewDecoratedNode(plan.DecorationTypeIndexedAccess, "Indexed table access on index [mytable2.i2]",
						plan.NewFilter(
							and(
								expression.NewEquals(
									expression.NewGetFieldWithTable(0, sql.Int32, "mytable2", "i2", true),
									expression.NewLiteral(21, sql.Int32),
								),
								expression.NewEquals(
									expression.NewGetFieldWithTable(2, sql.Int32, "mytable2", "t2", true),
									expression.NewLiteral("hello", sql.Text),
								),
							),
							plan.NewResolvedTable(table2.WithIndexLookup(
								mustIndexLookup(idxTable2I2.Get(21))),
							),
						),
					),
				),
			),
		},
		{
			name: "single index on aliased table",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "t1", "i", true),
				},
				plan.NewFilter(
					expression.NewEquals(
						expression.NewGetFieldWithTable(1, sql.Float64, "t1", "f", true),
						expression.NewLiteral(3.14, sql.Float64),
					),
					plan.NewTableAlias("t1",
						plan.NewResolvedTable(table),
					),
				),
			),
			expected: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "t1", "i", true),
				},
				plan.NewFilter(
					expression.NewEquals(
						expression.NewGetFieldWithTable(1, sql.Float64, "t1", "f", true),
						expression.NewLiteral(3.14, sql.Float64),
					),
					plan.NewTableAlias("t1",
						plan.NewDecoratedNode(plan.DecorationTypeIndexedAccess, "Indexed table access on index [mytable.f]",
							plan.NewResolvedTable(
								table.WithIndexLookup(
									mustIndexLookup(idxTable1F.Get(3.14)),
								),
							),
						),
					),
				),
			),
		},
		{
			name: "single index on aliased table, extra predicate",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "t1", "i", true),
				},
				plan.NewFilter(
					and(
						expression.NewEquals(
							expression.NewGetFieldWithTable(1, sql.Float64, "t1", "f", true),
							expression.NewLiteral(3.14, sql.Float64),
						),
						expression.NewEquals(
							expression.NewGetFieldWithTable(2, sql.Text, "t1", "t", true),
							expression.NewLiteral("hello", sql.Text),
						),
					),
					plan.NewTableAlias("t1",
						plan.NewResolvedTable(table),
					),
				),
			),
			expected: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "t1", "i", true),
				},
				plan.NewFilter(
					and(
						expression.NewEquals(
							expression.NewGetFieldWithTable(1, sql.Float64, "t1", "f", true),
							expression.NewLiteral(3.14, sql.Float64),
						),
						expression.NewEquals(
							expression.NewGetFieldWithTable(2, sql.Text, "t1", "t", true),
							expression.NewLiteral("hello", sql.Text),
						),
					),
					plan.NewTableAlias("t1",
						plan.NewDecoratedNode(plan.DecorationTypeIndexedAccess, "Indexed table access on index [mytable.f]",
							plan.NewResolvedTable(
								table.WithIndexLookup(
									mustIndexLookup(idxTable1F.Get(3.14)),
								),
							),
						),
					),
				),
			),
		},
		{
			name: "single index to each of two aliased tables",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "t1", "i", true),
				},
				plan.NewFilter(
					and(
						expression.NewEquals(
							expression.NewGetFieldWithTable(1, sql.Float64, "t1", "f", true),
							expression.NewLiteral(3.14, sql.Float64),
						),
						expression.NewEquals(
							expression.NewGetFieldWithTable(3, sql.Int32, "t2", "i2", true),
							expression.NewLiteral(21, sql.Int32),
						),
					),
					plan.NewCrossJoin(
						plan.NewTableAlias("t1",
							plan.NewResolvedTable(table),
						),
						plan.NewTableAlias("t2",
							plan.NewResolvedTable(table2),
						),
					),
				),
			),
			expected: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "t1", "i", true),
				},
				plan.NewCrossJoin(
					plan.NewFilter(
						expression.NewEquals(
							expression.NewGetFieldWithTable(1, sql.Float64, "t1", "f", true),
							expression.NewLiteral(3.14, sql.Float64),
						),
						plan.NewTableAlias("t1",
							plan.NewDecoratedNode(plan.DecorationTypeIndexedAccess, "Indexed table access on index [mytable.f]",
								plan.NewResolvedTable(table.WithIndexLookup(
									mustIndexLookup(idxTable1F.Get(3.14))),
								),
							),
						),
					),
					plan.NewFilter(
						expression.NewEquals(
							expression.NewGetFieldWithTable(0, sql.Int32, "t2", "i2", true),
							expression.NewLiteral(21, sql.Int32),
						),
						plan.NewTableAlias("t2",
							plan.NewDecoratedNode(plan.DecorationTypeIndexedAccess, "Indexed table access on index [mytable2.i2]",
								plan.NewResolvedTable(table2.WithIndexLookup(
									mustIndexLookup(idxTable2I2.Get(21))),
								),
							),
						),
					),
				),
			),
		},
		{
			name: "single index to each of two aliased tables, extra predicates",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "t1", "i", true),
				},
				plan.NewFilter(
					and(
						and(
							expression.NewEquals(
								expression.NewGetFieldWithTable(1, sql.Float64, "t1", "f", true),
								expression.NewLiteral(3.14, sql.Float64),
							),
							expression.NewEquals(
								expression.NewGetFieldWithTable(3, sql.Int32, "t2", "i2", true),
								expression.NewLiteral(21, sql.Int32),
							),
						),
						and(
							expression.NewEquals(
								expression.NewGetFieldWithTable(2, sql.Text, "t1", "t", true),
								expression.NewLiteral("hello", sql.Text),
							),
							expression.NewEquals(
								expression.NewGetFieldWithTable(5, sql.Text, "t2", "t2", true),
								expression.NewLiteral("goodbye", sql.Text),
							),
						),
					),
					plan.NewCrossJoin(
						plan.NewTableAlias("t1",
							plan.NewResolvedTable(table),
						),
						plan.NewTableAlias("t2",
							plan.NewResolvedTable(table2),
						),
					),
				),
			),
			expected: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "t1", "i", true),
				},
				plan.NewCrossJoin(
					plan.NewFilter(
						and(
							expression.NewEquals(
								expression.NewGetFieldWithTable(1, sql.Float64, "t1", "f", true),
								expression.NewLiteral(3.14, sql.Float64),
							),
							expression.NewEquals(
								expression.NewGetFieldWithTable(2, sql.Text, "t1", "t", true),
								expression.NewLiteral("hello", sql.Text),
							),
						),
						plan.NewTableAlias("t1",
							plan.NewDecoratedNode(plan.DecorationTypeIndexedAccess, "Indexed table access on index [mytable.f]",
								plan.NewResolvedTable(table.WithIndexLookup(
									mustIndexLookup(idxTable1F.Get(3.14))),
								),
							),
						),
					),
					plan.NewFilter(
						and(
							expression.NewEquals(
								expression.NewGetFieldWithTable(0, sql.Int32, "t2", "i2", true),
								expression.NewLiteral(21, sql.Int32),
							),
							expression.NewEquals(
								expression.NewGetFieldWithTable(2, sql.Text, "t2", "t2", true),
								expression.NewLiteral("goodbye", sql.Text),
							),
						),
						plan.NewTableAlias("t2",
							plan.NewDecoratedNode(plan.DecorationTypeIndexedAccess, "Indexed table access on index [mytable2.i2]",
								plan.NewResolvedTable(table2.WithIndexLookup(
									mustIndexLookup(idxTable2I2.Get(21))),
								),
							),
						),
					),
				),
			),
		},
		{
			name: "two aliased tables, indexed join",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "t1", "i", true),
				},
				plan.NewFilter(
					and(
						expression.NewEquals(
							expression.NewGetFieldWithTable(3, sql.Int32, "t2", "i2", true),
							expression.NewLiteral(21, sql.Int32),
						),
						and(
							expression.NewEquals(
								expression.NewGetFieldWithTable(0, sql.Text, "t1", "i", true),
								expression.NewLiteral(100, sql.Int32),
							),
							expression.NewEquals(
								expression.NewGetFieldWithTable(5, sql.Text, "t2", "t2", true),
								expression.NewLiteral("goodbye", sql.Text),
							),
						),
					),
					plan.NewIndexedJoin(
						plan.NewTableAlias("t1",
							plan.NewResolvedTable(table),
						),
						plan.NewTableAlias("t2",
							plan.NewResolvedTable(table2),
						),
						plan.JoinTypeInner,
						eq(gf(0, "mytable", "i"), gf(3, "mytable2", "i2")),
						[]sql.Expression{gf(0, "mytable", "i")},
						idxTable1F,
					),
				),
			),
			expected: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "t1", "i", true),
				},
				plan.NewIndexedJoin(
					plan.NewFilter(
						expression.NewEquals(
							expression.NewGetFieldWithTable(0, sql.Text, "t1", "i", true),
							expression.NewLiteral(100, sql.Int32),
						),
						plan.NewTableAlias("t1",
							plan.NewDecoratedNode(plan.DecorationTypeIndexedAccess, "Indexed table access on index [mytable.i]",
								plan.NewResolvedTable(
									table.WithIndexLookup(
										mustIndexLookup(idxtable1I.Get(100)),
									),
								),
							),
						),
					),
					plan.NewFilter(
						and(
							expression.NewEquals(
								expression.NewGetFieldWithTable(0, sql.Int32, "t2", "i2", true),
								expression.NewLiteral(21, sql.Int32),
							),
							expression.NewEquals(
								expression.NewGetFieldWithTable(2, sql.Text, "t2", "t2", true),
								expression.NewLiteral("goodbye", sql.Text),
							),
						),
						plan.NewTableAlias("t2",
							plan.NewResolvedTable(table2),
						),
					),
					plan.JoinTypeInner,
					eq(gf(0, "mytable", "i"), gf(3, "mytable2", "i2")),
					[]sql.Expression{gf(0, "mytable", "i")},
					idxTable1F,
				),
			),
		},
		{
			name: "two aliased tables, left indexed join",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "t1", "i", true),
				},
				plan.NewFilter(
					and(
						expression.NewEquals(
							expression.NewGetFieldWithTable(3, sql.Int32, "t2", "i2", true),
							expression.NewLiteral(21, sql.Int32),
						),
						and(
							expression.NewEquals(
								expression.NewGetFieldWithTable(0, sql.Text, "t1", "i", true),
								expression.NewLiteral(100, sql.Int32),
							),
							expression.NewEquals(
								expression.NewGetFieldWithTable(5, sql.Text, "t2", "t2", true),
								expression.NewLiteral("goodbye", sql.Text),
							),
						),
					),
					plan.NewIndexedJoin(
						plan.NewTableAlias("t1",
							plan.NewResolvedTable(table),
						),
						plan.NewTableAlias("t2",
							plan.NewResolvedTable(table2),
						),
						plan.JoinTypeLeft,
						eq(gf(0, "mytable", "i"), gf(3, "mytable2", "i2")),
						[]sql.Expression{gf(0, "mytable", "i")},
						idxTable1F,
					),
				),
			),
			expected: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "t1", "i", true),
				},
				plan.NewFilter(
					and(
						expression.NewEquals(
							expression.NewGetFieldWithTable(3, sql.Int32, "t2", "i2", true),
							expression.NewLiteral(21, sql.Int32),
						),
						expression.NewEquals(
							expression.NewGetFieldWithTable(5, sql.Text, "t2", "t2", true),
							expression.NewLiteral("goodbye", sql.Text),
						),
					),
					plan.NewIndexedJoin(
						plan.NewFilter(
							expression.NewEquals(
								expression.NewGetFieldWithTable(0, sql.Text, "t1", "i", true),
								expression.NewLiteral(100, sql.Int32),
							),
							plan.NewTableAlias("t1",
								plan.NewDecoratedNode(plan.DecorationTypeIndexedAccess, "Indexed table access on index [mytable.i]",
									plan.NewResolvedTable(
										table.WithIndexLookup(
											mustIndexLookup(idxtable1I.Get(100)),
										),
									),
								),
							),
						),
						plan.NewTableAlias("t2",
							plan.NewResolvedTable(table2),
						),
						plan.JoinTypeLeft,
						eq(gf(0, "mytable", "i"), gf(3, "mytable2", "i2")),
						[]sql.Expression{gf(0, "mytable", "i")},
						idxTable1F,
					),
				),
			),
		},
		{
			name: "two aliased tables, right indexed join",
			node: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(0, sql.Int32, "t1", "i", true),
				},
				plan.NewFilter(
					and(
						expression.NewEquals(
							expression.NewGetFieldWithTable(3, sql.Int32, "t2", "i2", true),
							expression.NewLiteral(21, sql.Int32),
						),
						and(
							expression.NewEquals(
								expression.NewGetFieldWithTable(0, sql.Text, "t1", "i", true),
								expression.NewLiteral(100, sql.Int32),
							),
							expression.NewEquals(
								expression.NewGetFieldWithTable(5, sql.Text, "t2", "t2", true),
								expression.NewLiteral("goodbye", sql.Text),
							),
						),
					),
					plan.NewIndexedJoin(
						plan.NewTableAlias("t2",
							plan.NewResolvedTable(table2),
						),
						plan.NewTableAlias("t1",
							plan.NewResolvedTable(table),
						),
						plan.JoinTypeRight,
						eq(gf(0, "mytable", "i"), gf(3, "mytable2", "i2")),
						[]sql.Expression{gf(0, "mytable", "i")},
						idxTable1F,
					),
				),
			),
			expected: plan.NewProject(
				[]sql.Expression{
					expression.NewGetFieldWithTable(3, sql.Int32, "t1", "i", true),
				},
				plan.NewFilter(
					expression.NewEquals(
						expression.NewGetFieldWithTable(3, sql.Text, "t1", "i", true),
						expression.NewLiteral(100, sql.Int32),
					),
					plan.NewIndexedJoin(
						plan.NewFilter(
							and(
								expression.NewEquals(
									expression.NewGetFieldWithTable(0, sql.Int32, "t2", "i2", true),
									expression.NewLiteral(21, sql.Int32),
								),
								expression.NewEquals(
									expression.NewGetFieldWithTable(2, sql.Text, "t2", "t2", true),
									expression.NewLiteral("goodbye", sql.Text),
								),
							),
							plan.NewTableAlias("t2",
								plan.NewDecoratedNode(plan.DecorationTypeIndexedAccess, "Indexed table access on index [mytable2.i2]",
									plan.NewResolvedTable(
										table2.WithIndexLookup(
											mustIndexLookup(idxTable2I2.Get(21)),
										),
									),
								),
							),
						),
						plan.NewTableAlias("t1",
							plan.NewResolvedTable(table),
						),
						plan.JoinTypeRight,
						eq(gf(0, "mytable", "i"), gf(3, "mytable2", "i2")),
						[]sql.Expression{gf(0, "mytable", "i")},
						idxTable1F,
					),
				),
			),
		},
	}

	runTestCases(t, sql.NewEmptyContext(), tests, a, getRule("pushdown_filters"))
}

func mustIndexLookup(lookup sql.IndexLookup, err error) sql.IndexLookup {
	if err != nil {
		panic(err)
	}
	return lookup
}
