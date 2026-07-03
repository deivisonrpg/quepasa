CREATE TABLE IF NOT EXISTS user_contexts (
	username TEXT NOT NULL,
	contextid TEXT NOT NULL,
	label TEXT NOT NULL DEFAULT '',
	enabled BOOLEAN NOT NULL DEFAULT 1,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (username, contextid)
);

CREATE INDEX IF NOT EXISTS idx_user_contexts_contextid ON user_contexts (contextid);
