import { uuid } from "@local/uuid";
import i18next from "i18next";
import { ValidationError } from "yup";
import { getCurrentUser } from "../authentication/session";
export const AllLanguages = {
    "aa": "Qafaraf",
    "ab": "Аҧсуа бызшәа Aƥsua bızšwa / Аҧсшәа Aƥsua",
    "ace": "بهسا اچيه",
    "ach": "Lwo",
    "ada": "Dangme",
    "ady": "Адыгэбзэ",
    "af": "Afrikaans",
    "afh": "El-Afrihili",
    "ain": "アイヌ・イタㇰ / Ainu-itak",
    "ak": "Akan",
    "ale": "Уна́ӈам тунуу́ / Унаӈан умсуу",
    "alt": "Алтайские языки",
    "am": "አማርኛ",
    "an": "Aragonés",
    "anp": "अंगोली",
    "ar": "العربية",
    "arp": "Hinónoʼeitíít",
    "arw": "Lokono",
    "as": "অসমীয়া",
    "ast": "Asturianu",
    "av": "Магӏарул мацӏ / Авар мацӏ",
    "awa": "अवधी",
    "ay": "Aymar aru",
    "az": "Azərbaycan dili / آذربایجان دیلی / Азәрбајҹан дили",
    "ba": "Башҡорт теле / Başqort tele",
    "bai": "Bamiléké",
    "bal": "بلوچی",
    "ban": "ᬪᬵᬱᬩᬮᬶ; ᬩᬲᬩᬮᬶ / Basa Bali",
    "bas": "Mbene / Ɓasaá",
    "be": "Беларуская мова / Беларуская мова",
    "bej": "Bidhaawyeet",
    "bem": "Chibemba",
    "bg": "български език bălgarski ezik",
    "bho": "भोजपुरी",
    "bin": "Ẹ̀dó",
    "bla": "ᓱᖽᐧᖿ",
    "bm": "ߓߊߡߊߣߊߣߞߊߣ",
    "bn": "বাংলা Bāŋlā",
    "bo": "བོད་སྐད་ Bodskad / ལྷ་སའི་སྐད་ Lhas'iskad",
    "br": "Brezhoneg",
    "bra": "ब्रजुरी",
    "bs": "bosanski",
    "bua": "буряад хэлэн",
    "bug": "ᨅᨔ ᨕᨘᨁᨗ",
    "byn": "ብሊና; ብሊን",
    "ca": "català / valencià",
    "cad": "Hasí:nay",
    "car": "Kari'nja",
    "ce": "Нохчийн мотт / نَاخچیین موٓتت / ნახჩიე მუოთთ",
    "ceb": "Sinugbuanong Binisayâ",
    "ch": "Finu' Chamoru",
    "chm": "марий йылме",
    "chn": "chinuk wawa / wawa / chinook lelang / lelang",
    "cho": "Chahta'",
    "chp": "ᑌᓀᓱᒼᕄᓀ (Dënesųłiné)",
    "chr": "ᏣᎳᎩ ᎦᏬᏂᎯᏍᏗ Tsalagi gawonihisdi",
    "chy": "Tsėhésenėstsestȯtse",
    "cnr": "crnogorski / црногорски",
    "co": "Corsu / Lingua corsa",
    "crh": "Къырымтатарджа / Къырымтатар тили / Ҡырымтатарҗа / Ҡырымтатар тили",
    "cs": "čeština; český jazyk",
    "csb": "Kaszëbsczi jãzëk",
    "cv": "Чӑвашла",
    "cy": "Cymraeg / y Gymraeg",
    "da": "dansk",
    "dak": "Dakhótiyapi / Dakȟótiyapi",
    "dar": "дарган мез",
    "de": "Deutsch",
    "den": "Dene K'e",
    "din": "Thuɔŋjäŋ",
    "doi": "डोगी / ڈوگرى",
    "dsb": "Dolnoserbski / dolnoserbšćina",
    "dv": "Dhivehi / ދިވެހިބަސް",
    "dyu": "Julakan",
    "dz": "རྫོང་ཁ་ Ĵoŋkha",
    "ee": "Eʋegbe",
    "el": "Νέα Ελληνικά Néa Ellêniká",
    "en": "English",
    "eo": "Esperanto",
    "es": "español / castellano",
    "et": "eesti keel",
    "eu": "euskara",
    "fa": "فارسی Fārsiy",
    "fat": "Mfantse / Fante / Fanti",
    "ff": "Fulfulde / Pulaar / Pular",
    "fi": "suomen kieli",
    "fil": "Wikang Filipino",
    "fj": "Na Vosa Vakaviti",
    "fo": "Føroyskt",
    "fon": "Fon gbè",
    "fr": "français",
    "frr": "Frasch / Fresk / Freesk / Friisk",
    "frs": "Oostfreesk / Plattdüütsk",
    "fur": "Furlan",
    "fy": "Frysk",
    "ga": "Gaeilge",
    "gaa": "Gã",
    "gay": "Basa Gayo",
    "gd": "Gàidhlig",
    "gil": "Taetae ni Kiribati",
    "gl": "galego",
    "gn": "Avañe'ẽ",
    "gor": "Bahasa Hulontalo",
    "gsw": "Schwiizerdütsch",
    "gu": "ગુજરાતી Gujarātī",
    "gv": "Gaelg / Gailck",
    "gwi": "Dinjii Zhu’ Ginjik",
    "ha": "Harshen Hausa / هَرْشَن",
    "hai": "X̱aat Kíl / X̱aadas Kíl / X̱aayda Kil / Xaad kil",
    "haw": "ʻŌlelo Hawaiʻi",
    "he": "עברית 'Ivriyþ",
    "hi": "हिन्दी Hindī",
    "hil": "Ilonggo",
    "hmn": "lus Hmoob / lug Moob / lol Hmongb / 𖬇𖬰𖬞 𖬌𖬣𖬵",
    "hr": "hrvatski",
    "hsb": "hornjoserbšćina",
    "ht": "kreyòl ayisyen",
    "hu": "magyar nyelv",
    "hup": "Na:tinixwe Mixine:whe'",
    "hy": "Հայերէն Hayerèn / Հայերեն Hayeren",
    "hz": "Otjiherero",
    "iba": "Jaku Iban",
    "id": "bahasa Indonesia",
    "ig": "Asụsụ Igbo",
    "ii": "ꆈꌠꉙ Nuosuhxop",
    "ik": "Iñupiaq",
    "ilo": "Pagsasao nga Ilokano / Ilokano",
    "inh": "ГӀалгӀай мотт",
    "is": "íslenska",
    "it": "italiano / lingua italiana",
    "iu": "Ịjọ",
    "ja": "日本語 Nihongo",
    "jbo": "la .lojban.",
    "jpr": "Dzhidi",
    "jrb": "عربية يهودية / ערבית יהודית",
    "jv": "ꦧꦱꦗꦮ / Basa Jawa",
    "ka": "ქართული Kharthuli",
    "kaa": "Qaraqalpaq tili / Қарақалпақ тили",
    "kab": "Tamaziɣt Taqbaylit / Tazwawt",
    "kac": "Jingpho",
    "kbd": "Адыгэбзэ (Къэбэрдейбзэ) Adıgăbză (Qăbărdeĭbză)",
    "kha": "কা কতিয়েন খাশি",
    "ki": "Gĩkũyũ",
    "kk": "қазақ тілі qazaq tili / қазақша qazaqşa",
    "km": "ភាសាខ្មែរ Phiəsaakhmær",
    "kn": "ಕನ್ನಡ Kannađa",
    "ko": "한국어 Han'gug'ô",
    "kok": "कोंकणी",
    "kpe": "Kpɛlɛwoo",
    "krc": "Къарачай-Малкъар тил / Таулу тил",
    "krl": "karjal / kariela / karjala",
    "kru": "कुड़ुख़",
    "ks": "कॉशुर / كأشُر",
    "ku": "kurdî / کوردی",
    "kum": "къумукъ тил / qumuq til",
    "kv": "Коми кыв",
    "kw": "Kernowek",
    "ky": "кыргызча kırgızça / кыргыз тили kırgız tili",
    "la": "Lingua latīna",
    "lad": "Judeo-español",
    "lah": "بھارت کا",
    "lb": "Lëtzebuergesch",
    "lez": "Лезги чӏал",
    "lg": "Luganda",
    "li": "Lèmburgs",
    "lo": "ພາສາລາວ Phasalaw",
    "lol": "Lomongo",
    "lt": "lietuvių kalba",
    "lu": "Kiluba",
    "lua": "Tshiluba",
    "lui": "Cham'teela",
    "lun": "Chilunda",
    "luo": "Dholuo",
    "lus": "Mizo ṭawng",
    "lv": "Latviešu valoda",
    "mad": "Madhura",
    "mag": "मगही",
    "mai": "मैथिली; মৈথিলী",
    "mak": "Basa Mangkasara' / ᨅᨔ ᨆᨀᨔᨑ",
    "man": "Mandi'nka kango",
    "mas": "ɔl",
    "mdf": "мокшень кяль",
    "men": "Mɛnde yia",
    "mga": "Gaoidhealg",
    "mh": "Kajin M̧ajeļ",
    "mi": "Te Reo Māori",
    "mic": "Míkmawísimk",
    "min": "Baso Minang",
    "mk": "македонски јазик makedonski jazik",
    "ml": "മലയാളം Malayāļã",
    "mn": "монгол хэл mongol xel / ᠮᠣᠩᠭᠣᠯ ᠬᠡᠯᠡ",
    "mnc": "ᠮᠠᠨᠵᡠ ᡤᡳᠰᡠᠨ Manju gisun",
    "moh": "Kanien’kéha",
    "mos": "Mooré",
    "mr": "मराठी Marāţhī",
    "ms": "Bahasa Melayu",
    "mt": "Malti",
    "mus": "Mvskoke",
    "mwl": "mirandés / lhéngua mirandesa",
    "mwr": "मारवाड़ी",
    "my": "မြန်မာစာ Mrãmācā / မြန်မာစကား Mrãmākā:",
    "na": "dorerin Naoero",
    "nap": "napulitano",
    "nb": "norsk bokmål",
    "nd": "siNdebele saseNyakatho",
    "nds": "Plattdütsch / Plattdüütsch",
    "ne": "नेपाली भाषा Nepālī bhāśā",
    "new": "नेपाल भाषा / नेवाः भाय्",
    "ng": "ndonga",
    "nia": "Li Niha",
    "niu": "ko e vagahau Niuē",
    "nl": "Nederlands; Vlaams",
    "nn": "norsk nynorsk",
    "no": "norsk",
    "nog": "Ногай тили",
    "nr": "isiNdebele seSewula",
    "nso": "Sesotho sa Leboa",
    "nub": "لغات نوبية",
    "nv": "Diné bizaad / Naabeehó bizaad",
    "ny": "Chichewa; Chinyanja",
    "nyo": "Runyoro",
    "oc": "occitan; lenga d'òc",
    "om": "Afaan Oromoo",
    "or": "ଓଡ଼ିଆ",
    "os": "Ирон ӕвзаг Iron ævzag",
    "osa": "Wazhazhe ie / 𐓏𐓘𐓻𐓘𐓻𐓟 𐒻𐓟",
    "pa": "ਪੰਜਾਬੀ / پنجابی Pãjābī",
    "pag": "Salitan Pangasinan",
    "pam": "Amánung Kapampangan / Amánung Sísuan",
    "pap": "Papiamentu",
    "pau": "a tekoi er a Belau",
    "pl": "Język polski",
    "ps": "پښتو Pax̌tow",
    "pt": "português",
    "qu": "Runa simi / kichwa simi / Nuna shimi",
    "raj": "राजस्थानी",
    "rap": "Vananga rapa nui",
    "rar": "Māori Kūki 'Āirani",
    "rm": "Rumantsch / Rumàntsch / Romauntsch / Romontsch",
    "rn": "Ikirundi",
    "ro": "limba română",
    "rom": "romani čhib",
    "ru": "русский язык russkiĭ âzık",
    "rup": "armãneashce / armãneashti / rrãmãneshti",
    "rw": "Ikinyarwanda",
    "sa": "संस्कृतम् Sąskŕtam / 𑌸𑌂𑌸𑍍𑌕𑍃𑌤𑌮𑍍",
    "sad": "Sandaweeki",
    "sah": "Сахалыы",
    "sam": "ארמית",
    "sat": "ᱥᱟᱱᱛᱟᱲᱤ",
    "sc": "sardu / limba sarda / lingua sarda",
    "scn": "Sicilianu",
    "sco": "Braid Scots; Lallans",
    "sd": "سنڌي / सिन्धी / ਸਿੰਧੀ",
    "se": "davvisámegiella",
    "sg": "yângâ tî sängö",
    "shn": "ၵႂၢမ်းတႆးယႂ်",
    "si": "සිංහල Sĩhala",
    "sid": "Sidaamu Afoo",
    "sk": "slovenčina / slovenský jazyk",
    "sl": "slovenski jezik / slovenščina",
    "sm": "Gagana faʻa Sāmoa",
    "sma": "Åarjelsaemien gïele",
    "smj": "julevsámegiella",
    "smn": "anarâškielâ",
    "sms": "sääʹmǩiõll",
    "sn": "chiShona",
    "snk": "Sooninkanxanne",
    "so": "af Soomaali",
    "sq": "Shqip",
    "sr": "српски / srpski",
    "srr": "Seereer",
    "ss": "siSwati",
    "st": "Sesotho [southern]",
    "su": "ᮘᮞ ᮞᮥᮔ᮪ᮓ / Basa Sunda",
    "suk": "Kɪsukuma",
    "sus": "Sosoxui",
    "sv": "svenska",
    "sw": "Kiswahili",
    "syr": "ܠܫܢܐ ܣܘܪܝܝܐ Lešānā Suryāyā",
    "ta": "தமிழ் Tamił",
    "te": "తెలుగు Telugu",
    "tem": "KʌThemnɛ",
    "ter": "Terêna",
    "tet": "Lia-Tetun",
    "tg": "тоҷикӣ toçikī",
    "th": "ภาษาไทย Phasathay",
    "ti": "ትግርኛ",
    "tig": "ትግረ / ትግሬ / ኻሳ / ትግራይት",
    "tk": "Türkmençe / Түркменче / تورکمن تیلی تورکمنچ; türkmen dili / түркмен дили",
    "tl": "Wikang Tagalog",
    "tli": "Lingít",
    "tn": "Setswana",
    "to": "lea faka-Tonga",
    "tog": "chiTonga",
    "tr": "Türkçe",
    "ts": "Xitsonga",
    "tt": "татар теле / tatar tele / تاتار",
    "tum": "chiTumbuka",
    "tvl": "Te Ggana Tuuvalu / Te Gagana Tuuvalu",
    "ty": "Reo Tahiti / Reo Mā'ohi",
    "tyv": "тыва дыл",
    "udm": "ئۇيغۇرچە / ئۇيغۇر تىلى",
    "ug": "Українська мова / Українська",
    "uk": "Úmbúndú",
    "ur": "اُردُو Urduw",
    "uz": "Oʻzbekcha / Ózbekça / ўзбекча / ئوزبېچه; oʻzbek tili / ўзбек тили / ئوبېک تیلی",
    "vai": "ꕙꔤ",
    "ve": "Tshivenḓa",
    "vi": "Tiếng Việt",
    "vot": "vađđa ceeli",
    "wa": "Walon",
    "war": "Winaray / Samareño / Lineyte-Samarnon / Binisayâ nga Winaray / Binisayâ nga Samar-Leyte / “Binisayâ nga Waray”",
    "was": "wá:šiw ʔítlu",
    "xal": "Хальмг келн / Xaľmg keln",
    "xh": "isiXhosa",
    "yi": "אידיש /  יידיש / ייִדיש / Yidiš",
    "yo": "èdè Yorùbá",
    "za": "Vahcuengh / 話僮",
    "zap": "Diidxazá/Dizhsa",
    "zen": "Tuḍḍungiyya",
    "zgh": "ⵜⴰⵎⴰⵣⵉⵖⵜ ⵜⴰⵏⴰⵡⴰⵢⵜ",
    "zh": "中文 Zhōngwén / 汉语 / 漢語 Hànyǔ",
    "zu": "isiZulu",
    "zun": "Shiwi'ma",
    "zza": "kirmanckî / dimilkî / kirdkî / zazakî",
};
const localeLoaders = {
    "af": () => import("date-fns/locale/af"),
    "ar-DZ": () => import("date-fns/locale/ar-DZ"),
    "ar-EG": () => import("date-fns/locale/ar-EG"),
    "ar-MA": () => import("date-fns/locale/ar-MA"),
    "ar-SA": () => import("date-fns/locale/ar-SA"),
    "ar-TN": () => import("date-fns/locale/ar-TN"),
    "ar": () => import("date-fns/locale/ar"),
    "az": () => import("date-fns/locale/az"),
    "be-tarask": () => import("date-fns/locale/be-tarask"),
    "be": () => import("date-fns/locale/be"),
    "bg": () => import("date-fns/locale/bg"),
    "bn": () => import("date-fns/locale/bn"),
    "bs": () => import("date-fns/locale/bs"),
    "ca": () => import("date-fns/locale/ca"),
    "cs": () => import("date-fns/locale/cs"),
    "cy": () => import("date-fns/locale/cy"),
    "da": () => import("date-fns/locale/da"),
    "de-AT": () => import("date-fns/locale/de-AT"),
    "de": () => import("date-fns/locale/de"),
    "el": () => import("date-fns/locale/el"),
    "en-AU": () => import("date-fns/locale/en-AU"),
    "en-CA": () => import("date-fns/locale/en-CA"),
    "en-GB": () => import("date-fns/locale/en-GB"),
    "en-IE": () => import("date-fns/locale/en-IE"),
    "en-IN": () => import("date-fns/locale/en-IN"),
    "en-NZ": () => import("date-fns/locale/en-NZ"),
    "en-US": () => import("date-fns/locale/en-US"),
    "en-ZA": () => import("date-fns/locale/en-ZA"),
    "eo": () => import("date-fns/locale/eo"),
    "es": () => import("date-fns/locale/es"),
    "et": () => import("date-fns/locale/et"),
    "eu": () => import("date-fns/locale/eu"),
    "fa-IR": () => import("date-fns/locale/fa-IR"),
    "fi": () => import("date-fns/locale/fi"),
    "fr-CA": () => import("date-fns/locale/fr-CA"),
    "fr-CH": () => import("date-fns/locale/fr-CH"),
    "fr": () => import("date-fns/locale/fr"),
    "fy": () => import("date-fns/locale/fy"),
    "gd": () => import("date-fns/locale/gd"),
    "gl": () => import("date-fns/locale/gl"),
    "gu": () => import("date-fns/locale/gu"),
    "he": () => import("date-fns/locale/he"),
    "hi": () => import("date-fns/locale/hi"),
    "hr": () => import("date-fns/locale/hr"),
    "ht": () => import("date-fns/locale/ht"),
    "hu": () => import("date-fns/locale/hu"),
    "hy": () => import("date-fns/locale/hy"),
    "id": () => import("date-fns/locale/id"),
    "is": () => import("date-fns/locale/is"),
    "it-CH": () => import("date-fns/locale/it-CH"),
    "it": () => import("date-fns/locale/it"),
    "ja-Hira": () => import("date-fns/locale/ja-Hira"),
    "ja": () => import("date-fns/locale/ja"),
    "ka": () => import("date-fns/locale/ka"),
    "kk": () => import("date-fns/locale/kk"),
    "km": () => import("date-fns/locale/km"),
    "kn": () => import("date-fns/locale/kn"),
    "ko": () => import("date-fns/locale/ko"),
    "lb": () => import("date-fns/locale/lb"),
    "lt": () => import("date-fns/locale/lt"),
    "lv": () => import("date-fns/locale/lv"),
    "mk": () => import("date-fns/locale/mk"),
    "mn": () => import("date-fns/locale/mn"),
    "ms": () => import("date-fns/locale/ms"),
    "mt": () => import("date-fns/locale/mt"),
    "nb": () => import("date-fns/locale/nb"),
    "nl-BE": () => import("date-fns/locale/nl-BE"),
    "nl": () => import("date-fns/locale/nl"),
    "nn": () => import("date-fns/locale/nn"),
    "oc": () => import("date-fns/locale/oc"),
    "pl": () => import("date-fns/locale/pl"),
    "pt-BR": () => import("date-fns/locale/pt-BR"),
    "pt": () => import("date-fns/locale/pt"),
    "ro": () => import("date-fns/locale/ro"),
    "ru": () => import("date-fns/locale/ru"),
    "sk": () => import("date-fns/locale/sk"),
    "sl": () => import("date-fns/locale/sl"),
    "sq": () => import("date-fns/locale/sq"),
    "sr-Latn": () => import("date-fns/locale/sr-Latn"),
    "sr": () => import("date-fns/locale/sr"),
    "sv": () => import("date-fns/locale/sv"),
    "ta": () => import("date-fns/locale/ta"),
    "te": () => import("date-fns/locale/te"),
    "th": () => import("date-fns/locale/th"),
    "tr": () => import("date-fns/locale/tr"),
    "ug": () => import("date-fns/locale/ug"),
    "uk": () => import("date-fns/locale/uk"),
    "uz-Cyrl": () => import("date-fns/locale/uz-Cyrl"),
    "uz": () => import("date-fns/locale/uz"),
    "vi": () => import("date-fns/locale/vi"),
    "zh-CN": () => import("date-fns/locale/zh-CN"),
    "zh-HK": () => import("date-fns/locale/zh-HK"),
    "zh-TW": () => import("date-fns/locale/zh-TW"),
};
export const loadLocale = async (locale) => {
    const loader = localeLoaders[locale] || localeLoaders["en-US"];
    const module = await loader();
    return module.default;
};
export const getTranslation = (obj, languages, showAny = true) => {
    if (!obj || !obj.translations)
        return {};
    for (const translation of obj.translations) {
        if (languages.includes(translation.language)) {
            return translation;
        }
    }
    if (showAny && obj.translations.length > 0)
        return obj.translations[0];
    return {};
};
export const updateTranslationFields = (obj, language, changes) => {
    let translationFound = false;
    const translations = [];
    for (const translation of obj?.translations ?? []) {
        if (translation.language === language) {
            translationFound = true;
            translations.push({
                ...translation,
                ...changes,
            });
        }
        else {
            translations.push(translation);
        }
    }
    if (!translationFound) {
        translations.push({
            id: uuid(),
            ...changes,
            language,
        });
    }
    return translations;
};
export const updateTranslation = (objectWithTranslation, translation) => {
    if (!objectWithTranslation.translations)
        return [];
    let translationFound = false;
    const translations = [];
    for (const existingTranslation of objectWithTranslation.translations) {
        if (existingTranslation.language === translation.language) {
            translations.push({ ...translation });
            translationFound = true;
        }
        else {
            translations.push(existingTranslation);
        }
    }
    if (!translationFound) {
        translations.push(translation);
    }
    return translations;
};
export const getLanguageSubtag = (language) => {
    if (!language)
        return "";
    const parts = language.split("-");
    return parts[0];
};
export const getUserLanguages = (session, useDefault = true) => {
    const { languages } = getCurrentUser(session);
    if (languages && languages.length > 0) {
        return languages.filter(Boolean).map(getLanguageSubtag);
    }
    if (navigator.language) {
        return [getLanguageSubtag(navigator.language)];
    }
    return useDefault ? ["en"] : [];
};
export const getUserLocale = (session) => {
    const userLanguages = getUserLanguages(session);
    const navigatorLanguages = [...navigator.languages, navigator.language].filter(Boolean);
    const findMatchingLocale = (languages) => {
        for (const language of languages) {
            const matchingLocales = Object.keys(localeLoaders).filter((locale) => locale.split("-")[0] === language.split("-")[0]);
            if (matchingLocales.length > 0) {
                const exactMatch = matchingLocales.find((locale) => locale === language);
                return exactMatch || matchingLocales[0];
            }
        }
        return undefined;
    };
    const navLanguagesInUserLanguages = navigatorLanguages.filter((language) => userLanguages.includes(language.split("-")[0]));
    return (findMatchingLocale(navLanguagesInUserLanguages) ||
        findMatchingLocale(userLanguages) ||
        findMatchingLocale(navigatorLanguages) ||
        "en-US");
};
export const getPreferredLanguage = (availableLanguages, userLanguages) => {
    for (const userLanguage of userLanguages) {
        if (availableLanguages.includes(userLanguage))
            return userLanguage;
    }
    return availableLanguages[0];
};
export const getTranslationData = (field, meta, language) => {
    if (!field?.value || !Array.isArray(field.value))
        return { error: undefined, index: -1, touched: undefined, value: undefined };
    const index = field.value.findIndex(t => t.language === language);
    const value = field.value[index];
    const touched = meta.touched?.[index];
    const error = meta.error?.[index];
    return { error, index, touched, value };
};
export const handleTranslationBlur = (field, meta, event, language) => {
    const { name: blurredField } = event.target;
    const touched = meta.touched;
    if (!touched || !touched[blurredField]) {
        field.onBlur({ ...event, target: { ...event.target, name: `${field.name}.${language}.${blurredField}` } });
    }
};
export const handleTranslationChange = (field, meta, helpers, event, language) => {
    const { name: changedField } = event.target;
    const { index, value: currentValue } = getTranslationData(field, meta, language);
    const newValue = {
        ...currentValue,
        [changedField]: event.target.value,
    };
    const newTranslations = field.value.map((translation, idx) => idx === index ? newValue : translation);
    helpers.setValue(newTranslations);
};
export const getFormikErrorsWithTranslations = (field, meta, validationSchema) => {
    const errors = {};
    for (const translation of field.value) {
        const subtag = translation.language;
        const language = AllLanguages[subtag] ?? subtag;
        try {
            validationSchema.validateSync(translation, { abortEarly: false });
        }
        catch (error) {
            if (error instanceof ValidationError) {
                for (const innerError of error.inner) {
                    const key = `${language} ${innerError.path}`;
                    errors[key] = innerError.errors;
                }
            }
        }
    }
    return errors;
};
export const combineErrorsWithTranslations = (errors, translationErrors) => {
    const combinedErrors = { ...errors, ...translationErrors };
    const filteredErrors = Object.fromEntries(Object.entries(combinedErrors).filter(([key]) => !key.startsWith("translations")));
    return filteredErrors;
};
export const addEmptyTranslation = (field, meta, helpers, language) => {
    const translations = [...field.value];
    const initialTranslations = Array.isArray(meta.initialValue) ? meta.initialValue : [];
    if (initialTranslations.length === 0) {
        console.error("Could not determine fields in translation object");
        return;
    }
    const newTranslation = { id: uuid(), language };
    for (const field of Object.keys(initialTranslations[0])) {
        if (!["id", "language"].includes(field))
            newTranslation[field] = "";
    }
    newTranslation.id = uuid();
    newTranslation.language = language;
    translations.push(newTranslation);
    helpers.setValue(translations);
};
export const removeTranslation = (field, meta, helpers, language) => {
    const translations = [...field.value];
    const { index } = getTranslationData(field, meta, language);
    translations.splice(index, 1);
    helpers.setValue(translations);
};
export const translateSnackMessage = (key, variables) => {
    const messageAsError = i18next.t(key, { ...variables, defaultValue: key, ns: "error" });
    const messageAsCommon = i18next.t(key, { ...variables, defaultValue: key, ns: "common" });
    if (messageAsError.length > 0 && messageAsError !== key) {
        const details = i18next.t(`${key}Details`, { ns: "error" });
        return { message: messageAsError, details: (details === `${key}Details` ? undefined : details) };
    }
    return { message: messageAsCommon, details: undefined };
};
export const getTranslatedTitleAndHelp = (data) => {
    if (!data)
        return {};
    let title = data.title;
    let help = data.help;
    if (!title && data.titleKey) {
        title = i18next.t(data.titleKey, { ...data.titleVariables, ns: "common", defaultValue: "" });
        if (title === "")
            title = undefined;
    }
    if (!help && data.helpKey) {
        help = i18next.t(data.helpKey, { ...data.helpVariables, ns: "common", defaultValue: "" });
        if (help === "")
            help = undefined;
    }
    return { title, help };
};
//# sourceMappingURL=translationTools.js.map