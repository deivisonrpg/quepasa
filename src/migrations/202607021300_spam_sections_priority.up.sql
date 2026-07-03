PRAGMA foreign_keys=OFF;

DROP TABLE IF EXISTS "spam_sections_migrated";

CREATE TABLE "spam_sections_migrated" (
  "token" CHAR (100) PRIMARY KEY NOT NULL REFERENCES "servers"("token") ON DELETE CASCADE,
  "priority" INTEGER NOT NULL DEFAULT 0,
  "enabled" BOOLEAN NOT NULL DEFAULT TRUE,
  "label" TEXT NOT NULL DEFAULT '',
  "created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT OR REPLACE INTO "spam_sections_migrated" (
  "token",
  "priority",
  "enabled",
  "label",
  "created_at",
  "updated_at"
)
SELECT * FROM "spam_sections";

DROP TABLE "spam_sections";
ALTER TABLE "spam_sections_migrated" RENAME TO "spam_sections";

CREATE INDEX IF NOT EXISTS "idx_spam_sections_priority" ON "spam_sections" ("priority", "token");
CREATE INDEX IF NOT EXISTS "idx_spam_sections_enabled" ON "spam_sections" ("enabled");

PRAGMA foreign_keys=ON;
