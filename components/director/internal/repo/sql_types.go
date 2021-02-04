package repo

import (
	"database/sql"
	"time"
)

// AsynchronousEntity denotes an DB-layer entity which can be timestamp with created_at, updated_at, deleted_at and ready values
type AsynchronousEntity interface {
	SetReady(ready bool)

	GetCreatedAt() time.Time
	SetCreatedAt(t time.Time)

	GetUpdatedAt() time.Time
	SetUpdatedAt(t time.Time)

	GetDeletedAt() time.Time
	SetDeletedAt(t time.Time)
}

func NewNullableString(text *string) sql.NullString {
	nullString := sql.NullString{}
	if text != nil {
		nullString.String = *text
		nullString.Valid = true
	}

	return nullString
}

func NewNullableInt(i *int) sql.NullInt32 {
	nullInt := sql.NullInt32{}
	if i != nil {
		nullInt.Int32 = int32(*i)
		nullInt.Valid = true
	}

	return nullInt
}

func NewValidNullableString(text string) sql.NullString {
	return sql.NullString{
		String: text,
		Valid:  true,
	}
}

func NewNullableBool(boolean *bool) sql.NullBool {
	var sqlBool sql.NullBool
	if boolean != nil {
		sqlBool = sql.NullBool{Valid: true, Bool: *boolean}
	}

	return sqlBool
}

func NewValidNullableBool(boolean bool) sql.NullBool {
	return sql.NullBool{
		Valid: true,
		Bool:  boolean,
	}
}

func StringPtrFromNullableString(sqlString sql.NullString) *string {
	if sqlString.Valid {
		return &sqlString.String
	}

	return nil
}

func IntPtrFromNullableInt(i sql.NullInt32) *int {
	if i.Valid {
		val := int(i.Int32)
		return &val
	}

	return nil
}

func BoolPtrFromNullableBool(sqlBool sql.NullBool) *bool {
	if sqlBool.Valid {
		return &sqlBool.Bool
	}
	return nil
}
