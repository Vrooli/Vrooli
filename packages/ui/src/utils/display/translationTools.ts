import { FormikErrors, FormikProps } from "formik";
import { ObjectSchema, ValidationError } from 'yup';
import { Session } from "types";
import { v4 as uuid } from 'uuid';

//TODO find and remove deprecated

/**
 * Array of all IANA language subtags, with their native names. 
 * Taken from https://en.wikipedia.org/wiki/List_of_ISO_639-2_codes and https://www.iana.org/assignments/language-subtag-registry/language-subtag-registry
 */
export const AllLanguages = {
    "aa": `Qafaraf`,
    "ab": `Аҧсуа бызшәа Aƥsua bızšwa / Аҧсшәа Aƥsua`,
    "ace": `بهسا اچيه`,
    "ach": `Lwo`,
    "ada": `Dangme`,
    "ady": `Адыгэбзэ`,
    "af": `Afrikaans`,
    "afh": `El-Afrihili`,
    "ain": `アイヌ・イタㇰ / Ainu-itak`,
    "ak": `Akan`,
    "ale": `Уна́ӈам тунуу́ / Унаӈан умсуу`,
    "alt": `Алтайские языки`,
    "am": `አማርኛ`,
    "an": `Aragonés`,
    "anp": `अंगोली`,
    "ar": `العربية`,
    "arp": `Hinónoʼeitíít`,
    "arw": `Lokono`,
    "as": `অসমীয়া`,
    "ast": `Asturianu`,
    "av": `Магӏарул мацӏ / Авар мацӏ`,
    "awa": `अवधी`,
    "ay": `Aymar aru`,
    "az": `Azərbaycan dili / آذربایجان دیلی / Азәрбајҹан дили`,
    "ba": `Башҡорт теле / Başqort tele`,
    "bai": `Bamiléké`,
    "bal": `بلوچی`,
    "ban": `ᬪᬵᬱᬩᬮᬶ; ᬩᬲᬩᬮᬶ / Basa Bali`,
    "bas": `Mbene / Ɓasaá`,
    "be": `Беларуская мова / Беларуская мова`,
    "bej": `Bidhaawyeet`,
    "bem": `Chibemba`,
    "bg": `български език bălgarski ezik`,
    "bho": `भोजपुरी`,
    "bin": `Ẹ̀dó`,
    "bla": `ᓱᖽᐧᖿ`,
    "bm": `ߓߊߡߊߣߊߣߞߊߣ`,
    "bn": `বাংলা Bāŋlā`,
    "bo": `བོད་སྐད་ Bodskad / ལྷ་སའི་སྐད་ Lhas'iskad`,
    "br": `Brezhoneg`,
    "bra": `ब्रजुरी`,
    "bs": `bosanski`,
    "bua": `буряад хэлэн`,
    "bug": `ᨅᨔ ᨕᨘᨁᨗ`,
    "byn": `ብሊና; ብሊን`,
    "ca": `català / valencià`,
    "cad": `Hasí:nay`,
    "car": `Kari'nja`,
    "ce": `Нохчийн мотт / نَاخچیین موٓتت / ნახჩიე მუოთთ`,
    "ceb": `Sinugbuanong Binisayâ`,
    "ch": `Finu' Chamoru`,
    "chm": `марий йылме`,
    "chn": `chinuk wawa / wawa / chinook lelang / lelang`,
    "cho": `Chahta'`,
    "chp": `ᑌᓀᓱᒼᕄᓀ (Dënesųłiné)`,
    "chr": `ᏣᎳᎩ ᎦᏬᏂᎯᏍᏗ Tsalagi gawonihisdi`,
    "chy": `Tsėhésenėstsestȯtse`,
    "cnr": `crnogorski / црногорски`,
    "co": `Corsu / Lingua corsa`,
    "crh": `Къырымтатарджа / Къырымтатар тили / Ҡырымтатарҗа / Ҡырымтатар тили`,
    "cs": `čeština; český jazyk`,
    "csb": `Kaszëbsczi jãzëk`,
    "cv": `Чӑвашла`,
    "cy": `Cymraeg / y Gymraeg`,
    "da": `dansk`,
    "dak": `Dakhótiyapi / Dakȟótiyapi`,
    "dar": `дарган мез`,
    "de": `Deutsch`,
    "den": `Dene K'e`,
    "din": `Thuɔŋjäŋ`,
    "doi": `डोगी / ڈوگرى`,
    "dsb": `Dolnoserbski / dolnoserbšćina`,
    "dv": `Dhivehi / ދިވެހިބަސް`,
    "dyu": `Julakan`,
    "dz": `རྫོང་ཁ་ Ĵoŋkha`,
    "ee": `Eʋegbe`,
    "el": `Νέα Ελληνικά Néa Ellêniká`,
    "en": `English`,
    "eo": `Esperanto`,
    "es": `español / castellano`,
    "et": `eesti keel`,
    "eu": `euskara`,
    "fa": `فارسی Fārsiy`,
    "fat": `Mfantse / Fante / Fanti`,
    "ff": `Fulfulde / Pulaar / Pular`,
    "fi": `suomen kieli`,
    "fil": `Wikang Filipino`,
    "fj": `Na Vosa Vakaviti`,
    "fo": `Føroyskt`,
    "fon": `Fon gbè`,
    "fr": `français`,
    "frr": `Frasch / Fresk / Freesk / Friisk`,
    "frs": `Oostfreesk / Plattdüütsk`,
    "fur": `Furlan`,
    "fy": `Frysk`,
    "ga": `Gaeilge`,
    "gaa": `Gã`,
    "gay": `Basa Gayo`,
    "gd": `Gàidhlig`,
    "gil": `Taetae ni Kiribati`,
    "gl": `galego`,
    "gn": `Avañe'ẽ`,
    "gor": `Bahasa Hulontalo`,
    "gsw": `Schwiizerdütsch`,
    "gu": `ગુજરાતી Gujarātī`,
    "gv": `Gaelg / Gailck`,
    "gwi": `Dinjii Zhu’ Ginjik`,
    "ha": `Harshen Hausa / هَرْشَن`,
    "hai": `X̱aat Kíl / X̱aadas Kíl / X̱aayda Kil / Xaad kil`,
    "haw": `ʻŌlelo Hawaiʻi`,
    "he": `עברית 'Ivriyþ`,
    "hi": `हिन्दी Hindī`,
    "hil": `Ilonggo`,
    "hmn": `lus Hmoob / lug Moob / lol Hmongb / 𖬇𖬰𖬞 𖬌𖬣𖬵`,
    "hr": `hrvatski`,
    "hsb": `hornjoserbšćina`,
    "ht": `kreyòl ayisyen`,
    "hu": `magyar nyelv`,
    "hup": `Na:tinixwe Mixine:whe'`,
    "hy": `Հայերէն Hayerèn / Հայերեն Hayeren`,
    "hz": `Otjiherero`,
    "iba": `Jaku Iban`,
    "id": `bahasa Indonesia`,
    "ig": `Asụsụ Igbo`,
    "ii": `ꆈꌠꉙ Nuosuhxop`,
    "ik": `Iñupiaq`,
    "ilo": `Pagsasao nga Ilokano / Ilokano`,
    "inh": `ГӀалгӀай мотт`,
    "is": `íslenska`,
    "it": `italiano / lingua italiana`,
    "iu": `Ịjọ`,
    "ja": `日本語 Nihongo`,
    "jbo": `la .lojban.`,
    "jpr": `Dzhidi`,
    "jrb": `عربية يهودية / ערבית יהודית`,
    "jv": `ꦧꦱꦗꦮ / Basa Jawa`,
    "ka": `ქართული Kharthuli`,
    "kaa": `Qaraqalpaq tili / Қарақалпақ тили`,
    "kab": `Tamaziɣt Taqbaylit / Tazwawt`,
    "kac": `Jingpho`,
    "kbd": `Адыгэбзэ (Къэбэрдейбзэ) Adıgăbză (Qăbărdeĭbză)`,
    "kha": `কা কতিয়েন খাশি`,
    "ki": `Gĩkũyũ`,
    "kk": `қазақ тілі qazaq tili / қазақша qazaqşa`,
    "km": `ភាសាខ្មែរ Phiəsaakhmær`,
    "kn": `ಕನ್ನಡ Kannađa`,
    "ko": `한국어 Han'gug'ô`,
    "kok": `कोंकणी`,
    "kpe": `Kpɛlɛwoo`,
    "krc": `Къарачай-Малкъар тил / Таулу тил`,
    "krl": `karjal / kariela / karjala`,
    "kru": `कुड़ुख़`,
    "ks": `कॉशुर / كأشُر`,
    "ku": `kurdî / کوردی`,
    "kum": `къумукъ тил / qumuq til`,
    "kv": `Коми кыв`,
    "kw": `Kernowek`,
    "ky": `кыргызча kırgızça / кыргыз тили kırgız tili`,
    "la": `Lingua latīna`,
    "lad": `Judeo-español`,
    "lah": `بھارت کا`,
    "lb": `Lëtzebuergesch`,
    "lez": `Лезги чӏал`,
    "lg": `Luganda`,
    "li": `Lèmburgs`,
    "lo": `ພາສາລາວ Phasalaw`,
    "lol": `Lomongo`,
    "lt": `lietuvių kalba`,
    "lu": `Kiluba`,
    "lua": `Tshiluba`,
    "lui": `Cham'teela`,
    "lun": `Chilunda`,
    "luo": `Dholuo`,
    "lus": `Mizo ṭawng`,
    "lv": `Latviešu valoda`,
    "mad": `Madhura`,
    "mag": `मगही`,
    "mai": `मैथिली; মৈথিলী`,
    "mak": `Basa Mangkasara' / ᨅᨔ ᨆᨀᨔᨑ`,
    "man": `Mandi'nka kango`,
    "mas": `ɔl`,
    "mdf": `мокшень кяль`,
    "men": `Mɛnde yia`,
    "mga": `Gaoidhealg`,
    "mh": `Kajin M̧ajeļ`,
    "mi": `Te Reo Māori`,
    "mic": `Míkmawísimk`,
    "min": `Baso Minang`,
    "mk": `македонски јазик makedonski jazik`,
    "ml": `മലയാളം Malayāļã`,
    "mn": `монгол хэл mongol xel / ᠮᠣᠩᠭᠣᠯ ᠬᠡᠯᠡ`,
    "mnc": `ᠮᠠᠨᠵᡠ ᡤᡳᠰᡠᠨ Manju gisun`,
    "moh": `Kanien’kéha`,
    "mos": `Mooré`,
    "mr": `मराठी Marāţhī`,
    "ms": `Bahasa Melayu`,
    "mt": `Malti`,
    "mus": `Mvskoke`,
    "mwl": `mirandés / lhéngua mirandesa`,
    "mwr": `मारवाड़ी`,
    "my": `မြန်မာစာ Mrãmācā / မြန်မာစကား Mrãmākā:`,
    "na": `dorerin Naoero`,
    "nap": `napulitano`,
    "nb": `norsk bokmål`,
    "nd": `siNdebele saseNyakatho`,
    "nds": `Plattdütsch / Plattdüütsch`,
    "ne": `नेपाली भाषा Nepālī bhāśā`,
    "new": `नेपाल भाषा / नेवाः भाय्`,
    "ng": `ndonga`,
    "nia": `Li Niha`,
    "niu": `ko e vagahau Niuē`,
    "nl": `Nederlands; Vlaams`,
    "nn": `norsk nynorsk`,
    "no": `norsk`,
    "nog": `Ногай тили`,
    "nr": `isiNdebele seSewula`,
    "nso": `Sesotho sa Leboa`,
    "nub": `لغات نوبية`,
    "nv": `Diné bizaad / Naabeehó bizaad`,
    "ny": `Chichewa; Chinyanja`,
    "nyo": `Runyoro`,
    "oc": `occitan; lenga d'òc`,
    "om": `Afaan Oromoo`,
    "or": `ଓଡ଼ିଆ`,
    "os": `Ирон ӕвзаг Iron ævzag`,
    "osa": `Wazhazhe ie / 𐓏𐓘𐓻𐓘𐓻𐓟 𐒻𐓟`,
    "pa": `ਪੰਜਾਬੀ / پنجابی Pãjābī`,
    "pag": `Salitan Pangasinan`,
    "pam": `Amánung Kapampangan / Amánung Sísuan`,
    "pap": `Papiamentu`,
    "pau": `a tekoi er a Belau`,
    "pl": `Język polski`,
    "ps": `پښتو Pax̌tow`,
    "pt": `português`,
    "qu": `Runa simi / kichwa simi / Nuna shimi`,
    "raj": `राजस्थानी`,
    "rap": `Vananga rapa nui`,
    "rar": `Māori Kūki 'Āirani`,
    "rm": `Rumantsch / Rumàntsch / Romauntsch / Romontsch`,
    "rn": `Ikirundi`,
    "ro": `limba română`,
    "rom": `romani čhib`,
    "ru": `русский язык russkiĭ âzık`,
    "rup": `armãneashce / armãneashti / rrãmãneshti`,
    "rw": `Ikinyarwanda`,
    "sa": `संस्कृतम् Sąskŕtam / 𑌸𑌂𑌸𑍍𑌕𑍃𑌤𑌮𑍍`,
    "sad": `Sandaweeki`,
    "sah": `Сахалыы`,
    "sam": `ארמית`,
    "sat": `ᱥᱟᱱᱛᱟᱲᱤ`,
    "sc": `sardu / limba sarda / lingua sarda`,
    "scn": `Sicilianu`,
    "sco": `Braid Scots; Lallans`,
    "sd": `سنڌي / सिन्धी / ਸਿੰਧੀ`,
    "se": `davvisámegiella`,
    "sg": `yângâ tî sängö`,
    "shn": `ၵႂၢမ်းတႆးယႂ်`,
    "si": `සිංහල Sĩhala`,
    "sid": `Sidaamu Afoo`,
    "sk": `slovenčina / slovenský jazyk`,
    "sl": `slovenski jezik / slovenščina`,
    "sm": `Gagana faʻa Sāmoa`,
    "sma": `Åarjelsaemien gïele`,
    "smj": `julevsámegiella`,
    "smn": `anarâškielâ`,
    "sms": `sääʹmǩiõll`,
    "sn": `chiShona`,
    "snk": `Sooninkanxanne`,
    "so": `af Soomaali`,
    "sq": `Shqip`,
    "sr": `српски / srpski`,
    "srr": `Seereer`,
    "ss": `siSwati`,
    "st": `Sesotho [southern]`,
    "su": `ᮘᮞ ᮞᮥᮔ᮪ᮓ / Basa Sunda`,
    "suk": `Kɪsukuma`,
    "sus": `Sosoxui`,
    "sv": `svenska`,
    "sw": `Kiswahili`,
    "syr": `ܠܫܢܐ ܣܘܪܝܝܐ Lešānā Suryāyā`,
    "ta": `தமிழ் Tamił`,
    "te": `తెలుగు Telugu`,
    "tem": `KʌThemnɛ`,
    "ter": `Terêna`,
    "tet": `Lia-Tetun`,
    "tg": `тоҷикӣ toçikī`,
    "th": `ภาษาไทย Phasathay`,
    "ti": `ትግርኛ`,
    "tig": `ትግረ / ትግሬ / ኻሳ / ትግራይት`,
    "tk": `Türkmençe / Түркменче / تورکمن تیلی تورکمنچ; türkmen dili / түркмен дили`,
    "tl": `Wikang Tagalog`,
    "tli": `Lingít`,
    "tn": `Setswana`,
    "to": `lea faka-Tonga`,
    "tog": `chiTonga`,
    "tr": `Türkçe`,
    "ts": `Xitsonga`,
    "tt": `татар теле / tatar tele / تاتار`,
    "tum": `chiTumbuka`,
    "tvl": `Te Ggana Tuuvalu / Te Gagana Tuuvalu`,
    "ty": `Reo Tahiti / Reo Mā'ohi`,
    "tyv": `тыва дыл`,
    "udm": `ئۇيغۇرچە / ئۇيغۇر تىلى`,
    "ug": `Українська мова / Українська`,
    "uk": `Úmbúndú`,
    "ur": `اُردُو Urduw`,
    "uz": `Oʻzbekcha / Ózbekça / ўзбекча / ئوزبېچه; oʻzbek tili / ўзбек тили / ئوبېک تیلی`,
    "vai": `ꕙꔤ`,
    "ve": `Tshivenḓa`,
    "vi": `Tiếng Việt`,
    "vot": `vađđa ceeli`,
    "wa": `Walon`,
    "war": `Winaray / Samareño / Lineyte-Samarnon / Binisayâ nga Winaray / Binisayâ nga Samar-Leyte / “Binisayâ nga Waray”`,
    "was": `wá:šiw ʔítlu`,
    "xal": `Хальмг келн / Xaľmg keln`,
    "xh": `isiXhosa`,
    "yi": `אידיש /  יידיש / ייִדיש / Yidiš`,
    "yo": `èdè Yorùbá`,
    "za": `Vahcuengh / 話僮`,
    "zap": `Diidxazá/Dizhsa`,
    "zen": `Tuḍḍungiyya`,
    "zgh": `ⵜⴰⵎⴰⵣⵉⵖⵜ ⵜⴰⵏⴰⵡⴰⵢⵜ`,
    "zh": `中文 Zhōngwén / 汉语 / 漢語 Hànyǔ`,
    "zu": `isiZulu`,
    "zun": `Shiwi'ma`,
    "zza": `kirmanckî / dimilkî / kirdkî / zazakî`,
};

/**
 * Retrieves a value from an object's translations
 * @param obj The object to retrieve the value from
 * @param field The field to retrieve the value from
 * @param languages The languages the user is requesting
 * @param showAny If true, will default to returning the first language if no value is found
 * @returns The value of the field in the object's translations
 */
export const getTranslation = <
    KeyField extends string,
    Translation extends { [key in KeyField]?: string | null | undefined } & { id: string, language: string },
    Obj extends { translations?: Translation[] | null | undefined }
>(obj: Obj | null | undefined, field: KeyField, languages: readonly string[], showAny: boolean = true): string | null | undefined => {
    if (!obj || !obj.translations) return undefined;
    // Loop through translations
    for (const translation of obj.translations) {
        // If this translation is one of the languages we're looking for, check for the field
        if (languages.includes(translation.language)) {
            if (translation[field]) return translation[field];
        }
    }
    if (showAny && obj.translations.length > 0) return obj.translations[0][field];
    // If we didn't find a translation, return undefined
    return undefined;
}

/**
 * Update a translation's key/value pairs for a specific language.
 * @param obj An object with a "translations" array
 * @param language The language to update
 * @param changes An object of key/value pairs to update
 * @returns Updated translations array
 */
export const updateTranslationFields = <
    Translation extends { id: string, language: string },
    Obj extends { translations: Translation[] | null | undefined }
>(
    obj: Obj | null | undefined,
    language: string,
    changes: { [key in string]?: string | null | undefined }
): Translation[] => {
    let translationFound = false;
    // Initialize new translations array
    let translations: Translation[] = []
    // Loop through translations
    for (let translation of obj?.translations ?? []) {
        // If language matches, update every field in changes. 
        // If an existing field is not in changes, keep it unchanged.
        // If a new field is not in the existing translation, add it.
        if (translation.language === language) {
            console.log('found translation. updating...', translation);
            translationFound = true;
            translations.push({
                ...translation,
                ...changes
            })
        }
        // Otherwise, keep the translation unchanged
        else {
            translations.push(translation);
        }
    }
    // If no translation was found, add a new one
    if (!translationFound) {
        console.log('no translation found, so adding new one...')
        translations.push({
            id: uuid(),
            ...changes,
            language
        } as Translation);
    }
    return translations;
}

/**
 * Update an entire translation object for a specific language.
 * @param objectWithTranslation An object with a "translations" array
 * @param translation The translation object to update, including at least the language code
 * @returns Updated translations array
 */
export const updateTranslation = <
    Translation extends { id: string, language: string },
    Obj extends { translations: Translation[] }
>(objectWithTranslation: Obj, translation: Translation): Translation[] => {
    if (!objectWithTranslation.translations) return [];
    let translationFound = false;
    let translations: Translation[] = []
    for (let existingTranslation of objectWithTranslation.translations) {
        if (existingTranslation.language === translation.language) {
            translations.push({ ...translation });
            translationFound = true;
        } else {
            translations.push(existingTranslation);
        }
    }
    if (!translationFound) {
        translations.push(translation);
    }
    return translations;
}

/**
 * Strips a language IETF code down to the subtag (e.g. en-US becomes en)
 * @param language IETF language code
 * @returns Subtag of language code
 */
export const getLanguageSubtag = (language: string): string => {
    if (!language) return "";
    const parts = language.split("-");
    return parts[0];
}

/**
 * Returns a list of user-preferred languages.
 * Priority order is the following: 
 * 1. Languages in session data
 * 2. Languages in browser (i.e. navigator.language)
 * 3. English
 * Strips languages so only the subtag is returned (e.g. en-US becomes en)
 * @param session Session data
 * @returns Array of user-preferred language subtags
 */
export const getUserLanguages = (session?: { languages?: Session['languages'] }): string[] => {
    if (session?.languages && session.languages.length > 0) {
        return session.languages.map(getLanguageSubtag);
    }
    if (navigator.language) {
        return [getLanguageSubtag(navigator.language)];
    }
    return ["en"];
}

/**
 * Finds the most preferred language in a list of languages.
 * @param availableLanguages The languages available
 * @param userLanguages The languages the user can speak, ordered by preference
 * @returns The most preferred language
 */
export const getPreferredLanguage = (availableLanguages: string[], userLanguages: string[]): string => {
    // Loop through user languages
    for (const userLanguage of userLanguages) {
        // If this language is available, return it
        if (availableLanguages.includes(userLanguage)) return userLanguage;
    }
    // If we didn't find a language, return the first available language
    return availableLanguages[0];
}

type TranslationObject = {
    id: string,
    language: string,
    [key: string]: string | null | undefined
}

/**
 * Finds the error, touched, and value for a translation field in a formik object
 * @param formik Formik object
 * @param field The field that contains the translation object array
 * @param language The language to retrieve
 * @returns The error, touched, and value for the translation object with the given language
 */
export const getTranslationData = <
    KeyField extends string,
    Values extends { [key in KeyField]: TranslationObject[] },
>(formik: FormikProps<Values>, field: KeyField, language: string): {
    error: FormikErrors<Values[KeyField][0]> | undefined,
    index: number,
    touched: { [key in keyof Values[KeyField][0]]: boolean } | undefined,
    value: Values[KeyField][0] | undefined
} => {
    if (!formik.values[field] || !Array.isArray(formik.values[field])) return { error: undefined, index: -1, touched: undefined, value: undefined };
    const index = formik.values[field].findIndex(t => t.language === language);
    console.log('translations', formik.values[field]);
    console.log('errors', formik.errors[field]);
    const value = formik.values[field][index];
    const touched = formik.touched[field]?.[index];
    const error = typeof formik.errors[field]?.[index] === 'object' ? formik.errors[field]?.[index] as any : undefined;
    return { error, index, touched, value };
}

/**
 * Handles onBlurs for translation fields in a formik object
 * @param formik Formik object
 * @param translationField The field that contains the translation object array
 * @param blurredField The field that was blurred
 * @param language The language to retrieve
 */
export const handleTranslationBlur = <
    KeyField extends string,
    Values extends { [key in KeyField]: TranslationObject[] },
>(formik: FormikProps<Values>, translationField: KeyField, event: React.FocusEvent<HTMLInputElement>, language: string) => {
    // Get field name from event
    const { name: blurredField } = event.target;
    // Check if field has already been touched
    const { index, touched } = getTranslationData(formik, translationField, language);
    // If not, set touched to true using dot notation
    if (!touched || !touched[blurredField]) {
        formik.setFieldTouched(`${translationField}.${index}.${String(blurredField)}`, true);
    }
}

/**
 * Handles onChange for translation fields in a formik object
 * @param formik Formik object
 * @param translationField The field that contains the translation object array
 * @param changedField The field that was changed
 * @param value The new value
 * @param language The language to retrieve
 */
export const handleTranslationChange = <
    KeyField extends string,
    Values extends { [key in KeyField]: TranslationObject[] },
>(formik: FormikProps<Values>, translationField: KeyField, event: React.ChangeEvent<HTMLInputElement>, language: string) => {
    // Get field name and value from event
    const { name: changedField, value } = event.target;
    // Get index of translation object
    const { index } = getTranslationData(formik, translationField, language);
    // Set the value using dot notation
    formik.setFieldValue(`${translationField}.${index}.${String(changedField)}`, value);
}

/**
 * Converts a formik error object into an error object which can be passed to GridSubmitButtons
 * @param formik Formik object
 * @param translationField The field that contains the translation object array
 * @param validationSchema The validation schema for a translation object
 * @returns An error object
 */
export const getFormikErrorsWithTranslations = <
    KeyField extends string,
    Values extends { [key in KeyField]: TranslationObject[] },
>(formik: FormikProps<Values>, translationField: KeyField, validationSchema: ObjectSchema<any>): { [key: string]: string | string[] } => {
    // Initialize errors object
    const errors: { [key: string]: string | string[] } = {};
    // Add all non-translation errors to errors object
    for (const [key, value] of Object.entries(formik.errors)) {
        if (typeof value === 'string' || (Array.isArray(value) && value.length > 0 && typeof value[0] === 'string')) {
            errors[key] = value;
        }
    }
    // Find translation errors. Since the given errors don't have the language subtag, we need to loop through all languages
    // and manually validate each field
    for (const translation of formik.values[translationField]) {
        const subtag = translation.language;
        // Get full name of language
        const language = AllLanguages[subtag] ?? subtag;
        // Manually validate translation object using validation schema
        try {
            validationSchema.validateSync(translation, { abortEarly: false });
        } catch (error) {
            // If validation error is thrown, use inner errors to add to errors object
            if (error instanceof ValidationError) {
                for (const innerError of error.inner) {
                    // Key should be full language name + field name (with first letter capitalized)
                    const key = `${language} ${innerError.path}`;
                    // Add error to errors object
                    errors[key] = innerError.errors;
                }
            }
        }
    }
    // Return errors object
    return errors;
}

/**
 * Adds a new, empty translation object (all fields '') to a formik translation field
 * @param formik Formik object
 * @param translationField The field that contains the translation object array
 * @param language The language to add
 */
export const addEmptyTranslation = <
    KeyField extends string,
    Values extends { [key in KeyField]: TranslationObject[] },
>(formik: FormikProps<Values>, translationField: KeyField, language: string) => {
    // Get copy of current translations
    const translations = [...formik.values[translationField]];
    // Determine fields in translation object (even if no translations exist yet). 
    // We can accomplish this through the initial values
    const initialTranslations = Array.isArray(formik.initialValues[translationField]) ? formik.initialValues[translationField] : [];
    if (initialTranslations.length === 0) {
        console.error('Could not determine fields in translation object');
        return;
    }
    // Create new translation object with all fields empty
    const newTranslation: TranslationObject = { id: uuid(), language };
    for (const field of Object.keys(initialTranslations[0])) {
        if (!['id', 'language'].includes(field)) newTranslation[field] = '';
    }
    newTranslation.id = uuid();
    newTranslation.language = language;
    // Add new translation object to translations
    translations.push(newTranslation);
    // Set new translations
    formik.setFieldValue(translationField, translations);
}

/**
 * Removes a translation object from a formik translation field
 * @param formik Formik object
 * @param translationField The field that contains the translation object array
 * @param language The language to remove
 */
export const removeTranslation = <
    KeyField extends string,
    Values extends { [key in KeyField]: TranslationObject[] },
>(formik: FormikProps<Values>, translationField: KeyField, language: string) => {
    // Get copy of current translations
    const translations = [...formik.values[translationField]];
    // Get index of translation object
    const { index } = getTranslationData(formik, translationField, language);
    // Remove translation object from translations
    translations.splice(index, 1);
    // Set new translations
    formik.setFieldValue(translationField, translations);
}