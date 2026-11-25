package lang

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fyne.io/fyne/v2"
)

func TestAddTranslations(t *testing.T) {
	err := AddTranslations(fyne.NewStaticResource("en.json", []byte(`{
		"Test": "Match"
	}`)))
	if assert.NoError(t, err) {
		setupLang("en")
		assert.Equal(t, "Match", L("Test"))
	}

	err = AddTranslationsForLocale([]byte(`{
		"Test2": "Match2"
	}`), "fr")
	if assert.NoError(t, err) {
		setupLang("fr")
		assert.Equal(t, "Match2", L("Test2"))
	}
}

func TestLocalize_Default(t *testing.T) {
	fallback := closestSupportedLocale([]string{"xx"})
	assert.Equal(t, fyne.Locale("en"), fallback[0:2])
}

func TestLocalize_Fallback(t *testing.T) {
	assert.Equal(t, "Missing", L("Missing"))
}

func TestLocalize_Struct(t *testing.T) {
	type data struct {
		Str string
	}

	assert.Equal(t, "Hello World!", L("Hello {{.Str}}!", data{Str: "World"}))
}

func TestLocalize_Map(t *testing.T) {
	assert.Equal(t, "Hello World!", L("Hello {{.Str}}!", map[string]string{"Str": "World"}))
}

func TestLocalizePlural_Fallback(t *testing.T) {
	require.NoError(t, AddTranslations(fyne.NewStaticResource("en.json", []byte(`{
		"Apple": {
			"one": "Apple",
			"other": "Apples"
		}
	}`))))

	setupLang("en")
	assert.Equal(t, "Missing", N("Missing", 1))
	assert.Equal(t, "Apple", N("Apple", 1))
	assert.Equal(t, "Apples", N("Apple", 2))
}

func TestLocalizeKey_Fallback(t *testing.T) {
	require.NoError(t, AddTranslations(fyne.NewStaticResource("en.json", []byte(`{
		"appleID": "Apple Matched"
	}`))))

	setupLang("en")
	assert.Equal(t, "Apple", X("appleIDMissing", "Apple"))
	assert.Equal(t, "Apple Matched", X("appleID", "Apple"))
}

func TestLocalizePluralKey_Fallback(t *testing.T) {
	require.NoError(t, AddTranslations(fyne.NewStaticResource("en.json", []byte(`{
		"appleID": {
			"one": "Apple",
			"other": "Apples"
		}
	}`))))

	setupLang("en")
	assert.Equal(t, "Missing", XN("appleIDMissing", "Missing", 1))
	assert.Equal(t, "Apple", XN("appleID", "Apple", 1))
	assert.Equal(t, "Apples", XN("appleID", "Apple", 2))
}

func TestLoadBuiltInTranslations(t *testing.T) {
	assert.NoError(t, AddTranslationsFS(translations, "translations"))
	setupLang("en")
	assert.Equal(t, "OK", X("OK", "FAIL"))
}

// An app that chose its language keeps Fyne's words in it, whatever the
// system's: French on a French computer until it chose English, English
// after, translations added later too; "" back to the system's.
func TestSetLanguage(t *testing.T) {
	t.Setenv("LANGUAGE", "fr")
	t.Setenv("LC_ALL", "fr_FR.UTF-8")
	t.Setenv("LANG", "fr_FR.UTF-8")
	defer SetLanguage("")

	SetLanguage("")
	assert.Equal(t, "Annuler", L("Cancel"), "the system's language: French")

	SetLanguage("en")
	assert.Equal(t, "Cancel", L("Cancel"), "the app chose English")
	require.NoError(t, AddTranslationsForLocale([]byte(`{"Later": "Plus tard"}`), "fr"))
	assert.Equal(t, "Cancel", L("Cancel"), "translations added later: still English")

	SetLanguage("de")
	assert.Equal(t, "Abbrechen", L("Cancel"), "another language chosen")
	SetLanguage("xx")
	assert.Equal(t, "Cancel", L("Cancel"), "a language with no translations: English")

	SetLanguage("")
	assert.Equal(t, "Annuler", L("Cancel"), "back to the system's")
}
