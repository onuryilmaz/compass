package mp_package

import (
	"database/sql"
	"encoding/json"
)

type Entity struct {
	ID                string          `db:"id"`
	TenantID          string          `db:"tenant_id"`
	ApplicationID     string          `db:"app_id"`
	OrdID             string          `db:"ord_id"`
	Vendor            sql.NullString  `db:"vendor"`
	Title             string          `db:"title"`
	ShortDescription  string          `db:"short_description"`
	Description       string          `db:"description"`
	Version           string          `db:"version"`
	PackageLinks      json.RawMessage `db:"package_links"`
	Links             json.RawMessage `db:"links"`
	LicenceType       sql.NullString  `db:"licence_type"`
	Tags              json.RawMessage `db:"tags"`
	Countries         json.RawMessage `db:"countries"`
	Labels            json.RawMessage `db:"labels"`
	PolicyLevel       string          `db:"policy_level"`
	CustomPolicyLevel sql.NullString  `db:"custom_policy_level"`
	PartOfProducts    json.RawMessage `db:"part_of_products"`
	LineOfBusiness    json.RawMessage `db:"line_of_business"`
	Industry          json.RawMessage `db:"industry"`
}
