import { IconButton, List, ListItem, Popover, Stack, TextField, Tooltip, Typography } from '@mui/material';
import { SelectLanguageDialogProps } from '../types';
import {
    ArrowDropDown as ArrowDropDownIcon,
    ArrowDropUp as ArrowDropUpIcon,
    Language as LanguageIcon,
} from '@mui/icons-material';
import { MouseEvent, useCallback, useMemo, useState } from 'react';
import { getUserLanguages } from 'utils';
import { FixedSizeList } from 'react-window';

/**
 * Array of all IANA language subtags, with their native names. 
 * Taken from https://en.wikipedia.org/wiki/List_of_ISO_639-2_codes and https://www.iana.org/assignments/language-subtag-registry/language-subtag-registry
 */
const AllLanguages = {
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

export const SelectLanguageDialog = ({
    availableLanguages,
    handleSelect,
    language,
    session,
}: SelectLanguageDialogProps) => {
    const [searchString, setSearchString] = useState('');
    const updateSearchString = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        setSearchString(event.target.value);
    }, []);

    const userLanguages = useMemo(() => getUserLanguages(session), [session]);
    const languageOptions = useMemo<Array<[string, string]>>(() => {
        // Handle restricted languages
        let options: Array<[string, string]> = availableLanguages ?
            availableLanguages.map(l => AllLanguages[l]) : Object.entries(AllLanguages);
        // Handle search string
        if (searchString.length > 0) {
            console.log('OPTIONS HEREEEE', options)
            options = options.filter((o: [string, string]) => o[1].toLowerCase().includes(searchString.toLowerCase()));
        }
        // Reorder so user's languages are first
        options = options.sort((a, b) => {
            const aIndex = userLanguages.indexOf(a[0]);
            const bIndex = userLanguages.indexOf(b[0]);
            if (aIndex === -1 && bIndex === -1) {
                return 0;
            } else if (aIndex === -1) {
                return 1;
            } else if (bIndex === -1) {
                return -1;
            } else {
                return aIndex - bIndex;
            }
        });
        return options;
    }, [availableLanguages, searchString, userLanguages]);

    // Popup for selecting language
    const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
    const open = Boolean(anchorEl);
    const onOpen = useCallback((event: MouseEvent<HTMLDivElement>) => setAnchorEl(event.currentTarget), []);
    const onClose = useCallback(() => setAnchorEl(null), []);

    return (
        <>
            {/* Language select popover */}
            <Popover
                open={open}
                anchorEl={anchorEl}
                onClose={onClose}
                sx={{
                    '& .MuiPopover-paper': {
                        background: 'transparent',
                        boxShadow: 'none',
                        border: 'none',
                        paddingBottom: 1,
                    }
                }}
                anchorOrigin={{
                    vertical: 'bottom',
                    horizontal: 'center',
                }}
                transformOrigin={{
                    vertical: 'top',
                    horizontal: 'center',
                }}
            >
                {/* Search bar and list of languages */}
                <Stack direction="column" spacing={2} sx={{
                    width: 'min(100vw, 400px)',
                    maxHeight: 'min(100vh, 600px)',
                    overflowX: 'auto',
                    overflowY: 'hidden',
                    background: (t) => t.palette.background.paper,
                    borderRadius: '8px',
                    padding: '8px',
                    "&::-webkit-scrollbar": {
                        width: 10,
                    },
                    "&::-webkit-scrollbar-track": {
                        backgroundColor: '#dae5f0',
                    },
                    "&::-webkit-scrollbar-thumb": {
                        borderRadius: '100px',
                        backgroundColor: "#409590",
                    },
                }}>
                    <TextField
                        placeholder="Enter language..."
                        autoFocus={true}
                        value={searchString}
                        onChange={updateSearchString}
                    />
                    <FixedSizeList
                        height={600}
                        width={384}
                        itemSize={46}
                        itemCount={languageOptions.length}
                        overscanCount={5}
                        style={{
                            scrollbarColor: '#409590 #dae5f0',
                            scrollbarWidth: 'thin',
                        }}
                    >
                        {(props) => {
                            const { index, style } = props;
                            const option: [string, string] = languageOptions[index];
                            return (
                                <ListItem
                                    key={index}
                                    style={style}
                                    disablePadding
                                    button
                                    onClick={() => { handleSelect(option[0]); onClose(); }}
                                >
                                    {option[1]}
                                </ListItem>
                            );
                        }}
                    </FixedSizeList>
                </Stack>
            </Popover>
            {/* Selected language label */}
            <Tooltip title="Select language" placement="top">
                <Stack direction="row" spacing={0} onClick={onOpen} sx={{
                    justifyContent: 'center',
                    alignItems: 'center',
                    borderRadius: '50px',
                    cursor: 'pointer',
                    background: '#4e7d31',
                    '&:hover': {
                        filter: 'brightness(120%)',
                    },
                    transition: 'all 0.2s ease-in-out',
                }}>
                    <IconButton size="large" sx={{ padding: '4px' }}>
                        <LanguageIcon sx={{ fill: 'white' }} />
                    </IconButton>
                    <Typography variant="body2">
                        {language.toLocaleUpperCase()}
                    </Typography>
                    {/* Drop down or drop up icon */}
                    <IconButton size="large" aria-label="language-select" sx={{ padding: '4px' }}>
                        {open ? <ArrowDropUpIcon sx={{ fill: 'white' }} /> : <ArrowDropDownIcon sx={{ fill: 'white' }} />}
                    </IconButton>
                </Stack>
            </Tooltip>
        </>
    )
}