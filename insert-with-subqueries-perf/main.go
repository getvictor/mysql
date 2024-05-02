package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"math/rand"
)

/*
Schema:

CREATE TABLE `teams` (
  `id` int(10) unsigned NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` varchar(1023) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `config` json DEFAULT NULL,
  `name_bin` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin GENERATED ALWAYS AS (`name`) VIRTUAL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name_bin` (`name_bin`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `policies` (
  `id` int(10) unsigned NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `team_id` int(10) unsigned DEFAULT NULL,
  `resolution` text COLLATE utf8mb4_unicode_ci,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `query` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `author_id` int(10) unsigned DEFAULT NULL,
  `platforms` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `critical` tinyint(1) NOT NULL DEFAULT '0',
  `checksum` binary(16) NOT NULL,
  `calendar_events_enabled` tinyint(1) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_policies_checksum` (`checksum`),
  KEY `idx_policies_author_id` (`author_id`),
  KEY `idx_policies_team_id` (`team_id`),
  CONSTRAINT `policies_ibfk_2` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `policy_membership` (
  `policy_id` int(10) unsigned NOT NULL,
  `host_id` int(10) unsigned NOT NULL,
  `passes` tinyint(1) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `automation_iteration` int(11) DEFAULT NULL,
  PRIMARY KEY (`policy_id`,`host_id`),
  KEY `idx_policy_membership_passes` (`passes`),
  KEY `idx_policy_membership_policy_id` (`policy_id`),
  KEY `idx_policy_membership_host_id_passes` (`host_id`,`passes`),
  CONSTRAINT `policy_membership_ibfk_1` FOREIGN KEY (`policy_id`) REFERENCES `policies` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `policy_stats` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `policy_id` int(10) unsigned NOT NULL,
  `inherited_team_id` int(10) unsigned NOT NULL DEFAULT '0',
  `passing_host_count` mediumint(8) unsigned NOT NULL DEFAULT '0',
  `failing_host_count` mediumint(8) unsigned NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `policy_team_unique` (`policy_id`,`inherited_team_id`),
  CONSTRAINT `policy_stats_ibfk_1` FOREIGN KEY (`policy_id`) REFERENCES `policies` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `hosts` (
  `id` int(10) unsigned NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `team_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `fk_hosts_team_id` (`team_id`),
  CONSTRAINT `hosts_ibfk_1` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

*/

// Create 10 teams
const teamCount = 10

// Create 10 global policies
const globalPolicyCount = 50

// Create 10 unique policies for each team
const teamPolicyCount = 20

// Create 10,000 global hosts
const globalHostCount = 10000

// Create 10,000 hosts for each team
const teamHostCount = 10000

// For each host and its policy, create a policy_membership record

func main() {
	db, err := sql.Open("mysql", "root:toor@tcp(127.0.0.1:3306)/test")
	panicIfErr(err)
	defer db.Close()

	_, err = db.Exec("DELETE FROM policy_stats WHERE id > 0")
	panicIfErr(err)
	_, err = db.Exec("DELETE FROM policy_membership WHERE host_id > 0")
	panicIfErr(err)
	_, err = db.Exec("DELETE FROM hosts WHERE id > 0")
	panicIfErr(err)
	_, err = db.Exec("DELETE FROM policies WHERE id > 0")
	panicIfErr(err)
	_, err = db.Exec("DELETE FROM teams WHERE id > 0")
	panicIfErr(err)

	// Create global policies
	for policyNumber := 1; policyNumber <= globalPolicyCount; policyNumber++ {
		name := fmt.Sprintf("global-policy-%d", policyNumber)
		_, err = db.Exec(
			fmt.Sprintf(
				"INSERT INTO policies (id, team_id, resolution, name, query, description, author_id, platforms, critical, checksum, calendar_events_enabled) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, %s, ?)",
				policiesChecksumComputedColumn(),
			),
			policyNumber, nil, "resolution", name, "query", "description", 1, "platforms", 1, 1,
		)
		panicIfErr(err)
	}
	log.Println("Created global policies")

	// Create global hosts
	for hostNumber := 1; hostNumber <= globalHostCount; hostNumber++ {
		_, err = db.Exec(
			"INSERT INTO hosts (id, team_id) VALUES (?, ?)", hostNumber, nil,
		)
		panicIfErr(err)
		// For each host, add a policy_membership record
		sqlStmt := "INSERT INTO policy_membership (policy_id, host_id, passes) VALUES "
		var args []interface{}
		for policyNumber := 1; policyNumber <= globalPolicyCount; policyNumber++ {
			var passes *bool
			val := rand.Int() % 100
			switch {
			case val < 10:
				passes = nil
			case val < 55:
				passes = new(bool)
				*passes = true
			default:
				passes = new(bool)
				*passes = false
			}
			sqlStmt += fmt.Sprintf("(?, ?, ?),")
			args = append(args, policyNumber, hostNumber, passes)
		}
		sqlStmt = sqlStmt[:len(sqlStmt)-1] // remove the trailing comma
		_, err = db.Exec(sqlStmt, args...)
		panicIfErr(err)
		if hostNumber%100 == 0 {
			log.Printf("Created %d global hosts", hostNumber)
		}
	}

	// Create teams
	for teamNumber := 1; teamNumber <= teamCount; teamNumber++ {
		_, err = db.Exec(
			"INSERT INTO teams (id, name, description) VALUES (?, ?, ?)", teamNumber, fmt.Sprintf("team-%d", teamNumber), fmt.Sprintf(
				"team-%d description",
				teamNumber,
			),
		)
		panicIfErr(err)
		// For each team create team policies
		for policyNumber := globalPolicyCount + (teamNumber-1)*teamPolicyCount + 1; policyNumber <= globalPolicyCount+teamNumber*teamPolicyCount; policyNumber++ {
			name := fmt.Sprintf("team-%d-policy-%d", teamNumber, policyNumber)
			_, err = db.Exec(
				fmt.Sprintf(
					"INSERT INTO policies (id, team_id, resolution, name, query, description, author_id, platforms, critical, checksum, calendar_events_enabled) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, %s, ?)",
					policiesChecksumComputedColumn(),
				),
				policyNumber, teamNumber, "resolution", name, "query", "description", 1, "platforms", 1, 1,
			)
			panicIfErr(err)
		}
		// For each team create team hosts
		for hostNumber := globalHostCount + (teamNumber-1)*teamHostCount + 1; hostNumber <= globalHostCount+teamNumber*teamHostCount; hostNumber++ {
			_, err = db.Exec(
				"INSERT INTO hosts (id, team_id) VALUES (?, ?)", hostNumber, teamNumber,
			)
			panicIfErr(err)
			// For each host, add a global policy_membership record
			sqlStmt := "INSERT INTO policy_membership (policy_id, host_id, passes) VALUES "
			var args []interface{}
			for policyNumber := 1; policyNumber <= globalPolicyCount; policyNumber++ {
				var passes *bool
				val := rand.Int() % 100
				switch {
				case val < 10:
					passes = nil
				case val < 55:
					passes = new(bool)
					*passes = true
				default:
					passes = new(bool)
					*passes = false
				}
				sqlStmt += fmt.Sprintf("(?, ?, ?),")
				args = append(args, policyNumber, hostNumber, passes)
			}
			sqlStmt = sqlStmt[:len(sqlStmt)-1] // remove the trailing comma
			_, err = db.Exec(sqlStmt, args...)
			panicIfErr(err)
			// For each host, add a team policy_membership record
			sqlStmt = "INSERT INTO policy_membership (policy_id, host_id, passes) VALUES "
			args = nil
			for policyNumber := globalPolicyCount + (teamNumber-1)*teamPolicyCount + 1; policyNumber <= globalPolicyCount+teamNumber*teamPolicyCount; policyNumber++ {
				var passes *bool
				val := rand.Int() % 100
				switch {
				case val < 10:
					passes = nil
				case val < 55:
					passes = new(bool)
					*passes = true
				default:
					passes = new(bool)
					*passes = false
				}
				sqlStmt += fmt.Sprintf("(?, ?, ?),")
				args = append(args, policyNumber, hostNumber, passes)
			}
			sqlStmt = sqlStmt[:len(sqlStmt)-1] // remove the trailing comma
			_, err = db.Exec(sqlStmt, args...)
			panicIfErr(err)
			if hostNumber%100 == 0 {
				log.Printf("Created %d hosts", hostNumber)
			}
		}
	}

	fmt.Println("Success!")
}

func panicIfErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func policiesChecksumComputedColumn() string {
	// concatenate with separator \x00
	return ` UNHEX(
		MD5(
			CONCAT_WS(CHAR(0),
				COALESCE(team_id, ''),
				name
			)
		)
	) `
}
