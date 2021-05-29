package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"
)

var emailEnabled = false
var messagesEnabled = false
var telegramEnabled = false
var discordEnabled = false
var matrixEnabled = false

func (app *appContext) GetPath(sect, key string) (fs.FS, string) {
	val := app.config.Section(sect).Key(key).MustString("")
	if strings.HasPrefix(val, "jfa-go:") {
		return localFS, strings.TrimPrefix(val, "jfa-go:")
	}
	dir, file := filepath.Split(val)
	return os.DirFS(dir), file
}

func (app *appContext) MustSetValue(section, key, val string) {
	app.config.Section(section).Key(key).SetValue(app.config.Section(section).Key(key).MustString(val))
}

func (app *appContext) loadConfig() error {
	var err error
	app.config, err = ini.Load(app.configPath)
	if err != nil {
		return err
	}

	app.MustSetValue("jellyfin", "public_server", app.config.Section("jellyfin").Key("server").String())

	for _, key := range app.config.Section("files").Keys() {
		if name := key.Name(); name != "html_templates" && name != "lang_files" {
			key.SetValue(key.MustString(filepath.Join(app.dataPath, (key.Name() + ".json"))))
		}
	}
	for _, key := range []string{"user_configuration", "user_displayprefs", "user_profiles", "ombi_template", "invites", "emails", "user_template", "custom_emails", "users", "telegram_users", "discord_users", "matrix_users"} {
		app.config.Section("files").Key(key).SetValue(app.config.Section("files").Key(key).MustString(filepath.Join(app.dataPath, (key + ".json"))))
	}
	app.URLBase = strings.TrimSuffix(app.config.Section("ui").Key("url_base").MustString(""), "/")
	app.config.Section("email").Key("no_username").SetValue(strconv.FormatBool(app.config.Section("email").Key("no_username").MustBool(false)))

	app.MustSetValue("password_resets", "email_html", "jfa-go:"+"email.html")
	app.MustSetValue("password_resets", "email_text", "jfa-go:"+"email.txt")

	app.MustSetValue("invite_emails", "email_html", "jfa-go:"+"invite-email.html")
	app.MustSetValue("invite_emails", "email_text", "jfa-go:"+"invite-email.txt")

	app.MustSetValue("email_confirmation", "email_html", "jfa-go:"+"confirmation.html")
	app.MustSetValue("email_confirmation", "email_text", "jfa-go:"+"confirmation.txt")

	app.MustSetValue("notifications", "expiry_html", "jfa-go:"+"expired.html")
	app.MustSetValue("notifications", "expiry_text", "jfa-go:"+"expired.txt")

	app.MustSetValue("notifications", "created_html", "jfa-go:"+"created.html")
	app.MustSetValue("notifications", "created_text", "jfa-go:"+"created.txt")

	app.MustSetValue("deletion", "email_html", "jfa-go:"+"deleted.html")
	app.MustSetValue("deletion", "email_text", "jfa-go:"+"deleted.txt")

	// Deletion template is good enough for these as well.
	app.MustSetValue("disable_enable", "disabled_html", "jfa-go:"+"deleted.html")
	app.MustSetValue("disable_enable", "disabled_text", "jfa-go:"+"deleted.txt")
	app.MustSetValue("disable_enable", "enabled_html", "jfa-go:"+"deleted.html")
	app.MustSetValue("disable_enable", "enabled_text", "jfa-go:"+"deleted.txt")

	app.MustSetValue("welcome_email", "email_html", "jfa-go:"+"welcome.html")
	app.MustSetValue("welcome_email", "email_text", "jfa-go:"+"welcome.txt")

	app.MustSetValue("template_email", "email_html", "jfa-go:"+"template.html")
	app.MustSetValue("template_email", "email_text", "jfa-go:"+"template.txt")

	app.MustSetValue("user_expiry", "behaviour", "disable_user")
	app.MustSetValue("user_expiry", "email_html", "jfa-go:"+"user-expired.html")
	app.MustSetValue("user_expiry", "email_text", "jfa-go:"+"user-expired.txt")

	app.config.Section("jellyfin").Key("version").SetValue(version)
	app.config.Section("jellyfin").Key("device").SetValue("jfa-go")
	app.config.Section("jellyfin").Key("device_id").SetValue(fmt.Sprintf("jfa-go-%s-%s", version, commit))
	messagesEnabled = app.config.Section("messages").Key("enabled").MustBool(false)
	telegramEnabled = app.config.Section("telegram").Key("enabled").MustBool(false)
	discordEnabled = app.config.Section("discord").Key("enabled").MustBool(false)
	matrixEnabled = app.config.Section("matrix").Key("enabled").MustBool(false)
	if !messagesEnabled {
		emailEnabled = false
		telegramEnabled = false
		discordEnabled = false
		matrixEnabled = false
	} else if app.config.Section("email").Key("method").MustString("") == "" {
		emailEnabled = false
	} else {
		emailEnabled = true
	}
	if !emailEnabled && !telegramEnabled && !discordEnabled && !matrixEnabled {
		messagesEnabled = false
	}

	app.MustSetValue("updates", "enabled", "true")
	releaseChannel := app.config.Section("updates").Key("channel").String()
	if app.config.Section("updates").Key("enabled").MustBool(false) {
		v := version
		if releaseChannel == "stable" {
			if version == "git" {
				v = "0.0.0"
			}
		} else if releaseChannel == "unstable" {
			v = "git"
		}
		app.updater = newUpdater(baseURL, namespace, repo, v, commit, updater)
	}
	if releaseChannel == "" {
		if version == "git" {
			releaseChannel = "unstable"
		} else {
			releaseChannel = "stable"
		}
		app.MustSetValue("updates", "channel", releaseChannel)
	}

	app.storage.customEmails_path = app.config.Section("files").Key("custom_emails").String()
	app.storage.loadCustomEmails()

	substituteStrings = app.config.Section("jellyfin").Key("substitute_jellyfin_strings").MustString("")

	oldFormLang := app.config.Section("ui").Key("language").MustString("")
	if oldFormLang != "" {
		app.storage.lang.chosenFormLang = oldFormLang
	}
	newFormLang := app.config.Section("ui").Key("language-form").MustString("")
	if newFormLang != "" {
		app.storage.lang.chosenFormLang = newFormLang
	}
	app.storage.lang.chosenAdminLang = app.config.Section("ui").Key("language-admin").MustString("en-us")
	app.storage.lang.chosenEmailLang = app.config.Section("email").Key("language").MustString("en-us")
	app.storage.lang.chosenPWRLang = app.config.Section("password_resets").Key("language").MustString("en-us")
	app.storage.lang.chosenTelegramLang = app.config.Section("telegram").Key("language").MustString("en-us")

	app.email = NewEmailer(app)

	return nil
}

func (app *appContext) migrateEmailConfig() {
	tempConfig, _ := ini.Load(app.configPath)
	fmt.Println(warning("Part of your email configuration will be migrated to the new \"messages\" section.\nA backup will be made."))
	err := tempConfig.SaveTo(app.configPath + "_" + commit + ".bak")
	if err != nil {
		app.err.Fatalf("Failed to backup config: %v", err)
		return
	}
	for _, setting := range []string{"use_24h", "date_format", "message"} {
		if val := app.config.Section("email").Key(setting).Value(); val != "" {
			tempConfig.Section("email").Key(setting).SetValue("")
			tempConfig.Section("messages").Key(setting).SetValue(val)
		}
	}
	if app.config.Section("messages").Key("enabled").MustBool(false) || app.config.Section("telegram").Key("enabled").MustBool(false) {
		tempConfig.Section("messages").Key("enabled").SetValue("true")
	}
	err = tempConfig.SaveTo(app.configPath)
	if err != nil {
		app.err.Fatalf("Failed to save config: %v", err)
		return
	}
	app.loadConfig()
}

func (app *appContext) migrateEmailStorage() error {
	var emails map[string]interface{}
	err := loadJSON(app.storage.emails_path, &emails)
	if err != nil {
		return err
	}
	newEmails := map[string]EmailAddress{}
	for jfID, addr := range emails {
		newEmails[jfID] = EmailAddress{
			Addr:    addr.(string),
			Contact: true,
		}
	}
	err = storeJSON(app.storage.emails_path+".bak", emails)
	if err != nil {
		return err
	}
	err = storeJSON(app.storage.emails_path, newEmails)
	if err != nil {
		return err
	}
	app.info.Println("Migrated to new email format. A backup has also been made.")
	return nil
}
