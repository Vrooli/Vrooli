import { Session } from ":local/consts";
import { CommonKey, ErrorKey } from ":local/translations";
import { uuid } from ":local/uuid";
import { FieldHelperProps, FieldInputProps, FieldMetaProps } from "formik";
import i18next from "i18next";
import { ObjectSchema, ValidationError } from "yup";
import { OptionalTranslation } from "../../types";
import { getCurrentUser } from "../authentication/session";

export type TranslationObject = {
    id: string,
    language: string,
    [key: string]: any,
}

/**
 * All supported IANA language subtags, with their native names. 
 * Taken from https://en.wikipedia.org/wiki/List_of_ISO_639-2_codes and https://www.iana.org/assignments/language-subtag-registry/language-subtag-registry
 */
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

type LocaleLoader = () => Promise<{ default: Locale }>;

/**
 * Maps date-fns locale codes to their corresponding loader functions. 
 * It would be nice if we could import like import('date-fns/locale/${locale}), 
 * but that doesn't work for some reason.
 */
const localeLoaders: Record<string, LocaleLoader> = {
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

export const loadLocale = async (locale: string): Promise<Locale> => {
    const loader = localeLoaders[locale] || localeLoaders["en-US"];
    const module = await loader();
    return module.default;
};

/**
 * Retrieves an object's translation for a given language code.
 * @param obj The object to retrieve the translation from.
 * @param languages The languages the user is requesting, in order of preference.
 * @param showAny If true, will default to returning the first language if no value is found
 * @returns The requested translation or an empty object if none is found
 */
export const getTranslation = <
    Translation extends { language: string },
>(obj: { translations?: Translation[] | null | undefined } | null | undefined, languages: readonly string[], showAny: boolean = true): Partial<Translation> => {
    if (!obj || !obj.translations) return {};
    // Loop through translations
    for (const translation of obj.translations) {
        // If this translation is one of the languages we're looking for
        if (languages.includes(translation.language)) {
            return translation;
        }
    }
    if (showAny && obj.translations.length > 0) return obj.translations[0];
    // If we didn't find a translation, return an empty object
    return {};
};

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
    changes: { [key in string]?: string | null | undefined },
): Translation[] => {
    let translationFound = false;
    // Initialize new translations array
    const translations: Translation[] = [];
    // Loop through translations
    for (const translation of obj?.translations ?? []) {
        // If language matches, update every field in changes. 
        // If an existing field is not in changes, keep it unchanged.
        // If a new field is not in the existing translation, add it.
        if (translation.language === language) {
            translationFound = true;
            translations.push({
                ...translation,
                ...changes,
            });
        }
        // Otherwise, keep the translation unchanged
        else {
            translations.push(translation);
        }
    }
    // If no translation was found, add a new one
    if (!translationFound) {
        translations.push({
            id: uuid(),
            ...changes,
            language,
        } as Translation);
    }
    return translations;
};

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
    const translations: Translation[] = [];
    for (const existingTranslation of objectWithTranslation.translations) {
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
};

/**
 * Strips a language IETF code down to the subtag (e.g. en-US becomes en)
 * @param language IETF language code
 * @returns Subtag of language code
 */
export const getLanguageSubtag = (language: string): string => {
    if (!language) return "";
    const parts = language.split("-");
    return parts[0];
};

/**
 * Returns a list of user-preferred languages.
 * Priority order is the following: 
 * 1. Languages in session data
 * 2. Languages in browser (i.e. navigator.language)
 * 3. English
 * Strips languages so only the subtag is returned (e.g. en-US becomes en)
 * @param session Session data
 * @param useDefault If true, will return English if no languages are found
 * @returns Array of user-preferred language subtags
 */
export const getUserLanguages = (session: Session | null | undefined, useDefault = true): string[] => {
    // First check session data for preferred languages
    const { languages } = getCurrentUser(session);
    if (languages && languages.length > 0) {
        return (languages.filter(Boolean) as string[]).map(getLanguageSubtag);
    }
    // If no languages are in session data, check browser
    if (navigator.language) {
        return [getLanguageSubtag(navigator.language)];
    }
    // Default to English if specified
    return useDefault ? ["en"] : [];
};

/**
 * Returns the best locale for the user based on their preferred languages.
 * Locale must be present in the DateFnsLocales list.
 */
export const getUserLocale = (session: Session | null | undefined): string => {
    const userLanguages = getUserLanguages(session);
    const navigatorLanguages = [...navigator.languages, navigator.language].filter(Boolean);

    const findMatchingLocale = (languages: string[]): string | undefined => {
        for (const language of languages) {
            const matchingLocales = Object.keys(localeLoaders).filter(
                (locale) => locale.split("-")[0] === language.split("-")[0],
            );
            if (matchingLocales.length > 0) {
                const exactMatch = matchingLocales.find((locale) => locale === language);
                return exactMatch || matchingLocales[0];
            }
        }
        return undefined;
    };

    const navLanguagesInUserLanguages = navigatorLanguages.filter((language) =>
        userLanguages.includes(language.split("-")[0]),
    );

    return (
        findMatchingLocale(navLanguagesInUserLanguages) ||
        findMatchingLocale(userLanguages) ||
        findMatchingLocale(navigatorLanguages) ||
        "en-US"
    );
};

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
};

/**
 * Finds the error, touched, and value for a translation field in a formik object
 * @param field The formik field that contains the translation object
 * @param meta The formik meta object for the field
 * @param language The language to retrieve
 * @returns The error, touched, and value for the translation object with the given language
 */
export const getTranslationData = <
    KeyField extends string,
    Values extends { [key in KeyField]: TranslationObject[] },
>(field: FieldInputProps<any>, meta: FieldMetaProps<any>, language: string): {
    error: { [key in keyof Values[KeyField][0]]: string } | undefined,
    index: number,
    touched: { [key in keyof Values[KeyField][0]]: boolean } | undefined,
    value: Values[KeyField][0] | undefined
} => {
    if (!field?.value || !Array.isArray(field.value)) return { error: undefined, index: -1, touched: undefined, value: undefined };
    const index = field.value.findIndex(t => t.language === language);
    const value = field.value[index];
    const touched = meta.touched?.[index];
    const error = meta.error?.[index] as any;
    return { error, index, touched, value };
};

/**
 * Handles onBlurs for translation fields in a formik object
 */
export const handleTranslationBlur = (
    field: FieldInputProps<{}>,
    meta: FieldMetaProps<{}>,
    event: { target: { name: string } },
    language: string,
) => {
    // Get field name from event
    const { name: blurredField } = event.target;
    // Check if field has already been touched
    const touched = meta.touched as any;
    // If not, set touched to true using dot notation
    if (!touched || !touched[blurredField]) {
        field.onBlur({ ...event, target: { ...event.target, name: `${field.name}.${language}.${blurredField}` } });
    }
};

/**
 * Handles onChange for translation fields in a formik object
 */
export const handleTranslationChange = (
    field: FieldInputProps<[]>,
    meta: FieldMetaProps<any>,
    helpers: FieldHelperProps<any>,
    event: { target: { name: string, value: string } },
    language: string,
) => {
    // Get field name and value from event
    const { name: changedField } = event.target;
    // Get index of translation object
    const { index, value: currentValue } = getTranslationData(field, meta, language);
    // Update the value of the translation object
    const newValue = {
        ...currentValue,
        [changedField]: event.target.value,
    };
    // Update the array with the new translation object
    const newTranslations = field.value.map((translation, idx) => idx === index ? newValue : translation);
    // Set the updated translations array
    helpers.setValue(newTranslations);
};

/**
 * Converts a formik error object into an error object which can be passed to GridSubmitButtons
 * @returns An error object
 */
export const getFormikErrorsWithTranslations = (
    field: FieldInputProps<any>,
    meta: FieldMetaProps<any>,
    validationSchema: ObjectSchema<any>,
): { [key: string]: string | string[] } => {
    // Initialize errors object
    const errors: { [key: string]: string | string[] } = {};
    // Find translation errors. Since the given errors don't have the language subtag, we need to loop through all languages
    // and manually validate each field
    for (const translation of field.value as any) {
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
};

/**
 * Combines normal errors object with translation errors object. 
 * Filter out any normal errors that start with "translations".
 * @param errors The normal errors object
 * @param translationErrors The translation errors object
 * @returns The combined errors object
 */
export const combineErrorsWithTranslations = (
    errors: { [key: string]: any },
    translationErrors: { [key: string]: any },
): { [key: string]: any } => {
    // Combine errors objects
    const combinedErrors = { ...errors, ...translationErrors };
    // Filter out any errors that start with "translations"
    const filteredErrors = Object.fromEntries(Object.entries(combinedErrors).filter(([key]) => !key.startsWith("translations")));
    // Return filtered errors
    return filteredErrors;
};

/**
 * Adds a new, empty translation object (all fields '') to a formik translation field
 */
export const addEmptyTranslation = (
    field: FieldInputProps<any>,
    meta: FieldMetaProps<any>,
    helpers: FieldHelperProps<{}>,
    language: string,
) => {
    // Get copy of current translations
    const translations = [...(field.value as any)];
    // Determine fields in translation object (even if no translations exist yet). 
    // We can accomplish this through the initial values
    const initialTranslations = Array.isArray(meta.initialValue) ? meta.initialValue : [];
    if (initialTranslations.length === 0) {
        console.error("Could not determine fields in translation object");
        return;
    }
    // Create new translation object with all fields empty
    const newTranslation: TranslationObject = { id: uuid(), language };
    for (const field of Object.keys(initialTranslations[0])) {
        if (!["id", "language"].includes(field)) newTranslation[field] = "";
    }
    newTranslation.id = uuid();
    newTranslation.language = language;
    // Add new translation object to translations
    translations.push(newTranslation);
    // Set new translations
    helpers.setValue(translations);
};

/**
 * Removes a translation object from a formik translation field
 */
export const removeTranslation = (
    field: FieldInputProps<any>,
    meta: FieldMetaProps<any>,
    helpers: FieldHelperProps<{}>,
    language: string,
) => {
    // Get copy of current translations
    const translations = [...(field.value as any)];
    // Get index of translation object
    const { index } = getTranslationData(field, meta, language);
    // Remove translation object from translations
    translations.splice(index, 1);
    // Set new translations
    helpers.setValue(translations);
};

/**
 * Converts a snack message code into a snack message and details. 
 * For now, details are only used for some errors
 * @param key The key to convert
 * @param variables The variables to use for translation
 * @returns Object with message and details
 */
export const translateSnackMessage = (
    key: ErrorKey | CommonKey,
    variables: { [x: string]: number | string } | undefined,
): { message: string, details: string | undefined } => {
    const messageAsError = i18next.t(key as ErrorKey, { ...variables, defaultValue: key, ns: "error" });
    const messageAsCommon = i18next.t(key as CommonKey, { ...variables, defaultValue: key, ns: "common" });
    if (messageAsError.length > 0 && messageAsError !== key) {
        const details = i18next.t(`${key}Details` as ErrorKey, { ns: "error" });
        return { message: messageAsError, details: (details === `${key}Details` ? undefined : details) };
    }
    return { message: messageAsCommon, details: undefined };
};

/**
 * Finds the translated title and help text for a component
 * @param data Data required to find the title and help text
 * @returns Object with title and help text, each of which can be undefined
 */
export const getTranslatedTitleAndHelp = (data: OptionalTranslation | null | undefined): { title?: string, help?: string } => {
    if (!data) return {};
    let title: string | undefined = data.title;
    let help: string | undefined = data.help;
    if (!title && data.titleKey) {
        title = i18next.t(data.titleKey, { ...data.titleVariables, ns: "common", defaultValue: "" });
        if (title === "") title = undefined;
    }
    if (!help && data.helpKey) {
        help = i18next.t(data.helpKey, { ...data.helpVariables, ns: "common", defaultValue: "" });
        if (help === "") help = undefined;
    }
    return { title, help };
};
