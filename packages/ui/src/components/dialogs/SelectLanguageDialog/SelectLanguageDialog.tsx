import { IconButton, List, ListItem, Popover, Stack, TextField, Tooltip, Typography } from '@mui/material';
import { SelectLanguageDialogProps } from '../types';
import {
    ArrowDropDown as ArrowDropDownIcon,
    ArrowDropUp as ArrowDropUpIcon,
    Language as LanguageIcon,
} from '@mui/icons-material';
import { MouseEvent, useCallback, useMemo, useState } from 'react';

/**
 * Array of all ISO 639-2 language codes, with their native names. 
 * Taken from https://en.wikipedia.org/wiki/List_of_ISO_639-2_codes
 */
const AllLanguages = {
    'aar': `Qafaraf`,
    'abk': `Аҧсуа бызшәа`,
    'ace': `بهسا اچيه`,
    'ach': `Lwo`,
    'ada': `Dangme`,
    'ady': `Адыгэбзэ`,
    'afh': `El-Afrihili`,
    'afr': `Afrikaans`,
    'ain': `アイヌ・イタㇰ / Ainu-itak`,
    'aka': `Akan`,
    'alb': `Shqip`,
    'ale': `Уна́ӈам тунуу́ / Унаӈан умсуу`,
    'alt': `Алтайские языки`,
    'amh': `አማርኛ`,
    'anp': `अंगोली`,
    'ara': `العربية`,
    'arg': `Aragonés`,
    'arm': `Հայերեն`,
    'arp': `Hinónoʼeitíít`,
    'arw': `Lokono`,
    'asm': `অসমীয়া`,
    'ast': `Asturianu`,
    'ava': `Магӏарул мацӏ / Авар мацӏ`,
    'awa': `अवधी`,
    'aym': `Aymar aru`,
    'aze': `Azərbaycan dili / آذربایجان دیلی / Азәрбајҹан дили`,
    'bai': `Bamiléké`,
    'bak': `Башҡорт теле / Başqort tele`,
    'bal': `بلوچی`,
    'bam': `ߓߊߡߊߣߊߣߞߊߣ`,
    'ban': `ᬪᬵᬱᬩᬮᬶ; ᬩᬲᬩᬮᬶ / Basa Bali`,
    'bas': `Mbene / Ɓasaá`,
    'bej': `Bidhaawyeet`,
    'bel': `Беларуская мова / Беларуская мова`,
    'bem': `Chibemba`,
    'ben': `বাংলা Bāŋlā`,
    'bho': `भोजपुरी`,
    'bin': `Ẹ̀dó`,
    'bla': `ᓱᖽᐧᖿ`,
    'bod': `བོད་སྐད་ Bodskad / ལྷ་སའི་སྐད་ Lhas'iskad`,
    'bos': `bosanski`,
    'bra': `ब्रजुरी`,
    'bre': `Brezhoneg`,
    'bua': `буряад хэлэн`,
    'bug': `ᨅᨔ ᨕᨘᨁᨗ`,
    'bul': `български език bălgarski ezik`,
    'byn': `ብሊና; ብሊን`,
    'cad': `Hasí:nay`,
    'car': `Kari'nja`,
    'cat': `català,valencià`,
    'ceb': `Sinugbuanong Binisayâ`,
    'ces': `čeština; český jazyk`,
    'cha': `Finu' Chamoru`,
    'che': `Нохчийн мотт / نَاخچیین موٓتت / ნახჩიე მუოთთ`,
    'chm': `марий йылме`,
    'chn': `chinuk wawa / wawa / chinook lelang / lelang`,
    'cho': `Chahta'`,
    'chp': `ᑌᓀᓱᒼᕄᓀ (Dënesųłiné)`,
    'chr': `ᏣᎳᎩ ᎦᏬᏂᎯᏍᏗ Tsalagi gawonihisdi`,
    'chv': `Чӑвашла`,
    'chy': `Tsėhésenėstsestȯtse`,
    'cnr': `crnogorski / црногорски`,
    'cor': `Kernowek`,
    'cos': `Corsu / Lingua corsa`,
    'crh': `Къырымтатарджа / Къырымтатар тили / Ҡырымтатарҗа / Ҡырымтатар тили`,
    'csb': `Kaszëbsczi jãzëk`,
    'cym': `Cymraeg / y Gymraeg`,
    'dak': `Dakhótiyapi / Dakȟótiyapi`,
    'dan': `dansk`,
    'dar': `дарган мез`,
    'den': `Dene K'e`,
    'deu': `Deutsch`,
    'din': `Thuɔŋjäŋ`,
    'div': `Dhivehi / ދިވެހިބަސް`,
    'doi': `डोगी / ڈوگرى`,
    'dsb': `Dolnoserbski / dolnoserbšćina`,
    'dyu': `Julakan`,
    'dzo': `རྫོང་ཁ་ Ĵoŋkha`,
    'ell': `Νέα Ελληνικά Néa Ellêniká`,
    'eng': `English`,
    'epo': `Esperanto`,
    'est': `eesti keel`,
    'eus': `euskara`,
    'ewe': `Eʋegbe`,
    'fao': `Føroyskt`,
    'fas': `فارسی Fārsiy`,
    'fat': `Mfantse / Fante / Fanti`,
    'fij': `Na Vosa Vakaviti`,
    'fil': `Wikang Filipino`,
    'fin': `suomen kieli`,
    'fon': `Fon gbè`,
    'fra': `français`,
    'frr': `Frasch / Fresk / Freesk / Friisk`,
    'frs': `Oostfreesk / Plattdüütsk`,
    'fry': `Frysk`,
    'ful': `Fulfulde / Pulaar / Pular`,
    'fur': `Furlan`,
    'gaa': `Gã`,
    'gay': `Basa Gayo`,
    'gil': `Taetae ni Kiribati`,
    'gla': `Gàidhlig`,
    'gle': `Gaeilge`,
    'glg': `galego`,
    'glv': `Gaelg / Gailck`,
    'gor': `Bahasa Hulontalo`,
    'grn': `Avañe'ẽ`,
    'gsw': `Schwiizerdütsch`,
    'guj': `ગુજરાતી Gujarātī`,
    'gwi': `Dinjii Zhu’ Ginjik`,
    'hai': `X̱aat Kíl / X̱aadas Kíl / X̱aayda Kil / Xaad kil`,
    'hat': `kreyòl ayisyen`,
    'hau': `Harshen Hausa / هَرْشَن`,
    'haw': `ʻŌlelo Hawaiʻi`,
    'heb': `עברית 'Ivriyþ`,
    'her': `Otjiherero`,
    'hil': `Ilonggo`,
    'hin': `हिन्दी Hindī`,
    'hmn': `lus Hmoob / lug Moob / lol Hmongb / 𖬇𖬰𖬞 𖬌𖬣𖬵`,
    'hrv': `hrvatski`,
    'hsb': `hornjoserbšćina`,
    'hun': `magyar nyelv`,
    'hup': `Na:tinixwe Mixine:whe'`,
    'hye': `Հայերէն Hayerèn / Հայերեն Hayeren`,
    'iba': `Jaku Iban`,
    'ibo': `Asụsụ Igbo`,
    'iii': `ꆈꌠꉙ Nuosuhxop`,
    'iku': `Ịjọ`,
    'ilo': `Pagsasao nga Ilokano / Ilokano`,
    'ind': `bahasa Indonesia`,
    'inh': `ГӀалгӀай мотт`,
    'ipk': `Iñupiaq`,
    'isl': `íslenska`,
    'ita': `italiano / lingua italiana`,
    'jav': `ꦧꦱꦗꦮ / Basa Jawa`,
    'jbo': `la .lojban.`,
    'jpn': `日本語 Nihongo`,
    'jpr': `Dzhidi`,
    'jrb': `عربية يهودية / ערבית יהודית`,
    'kaa': `Qaraqalpaq tili / Қарақалпақ тили`,
    'kab': `Tamaziɣt Taqbaylit / Tazwawt`,
    'kac': `Jingpho`,
    'kan': `ಕನ್ನಡ Kannađa`,
    'kas': `कॉशुर / كأشُر`,
    'kat': `ქართული Kharthuli`,
    'kaz': `қазақ тілі qazaq tili / қазақша qazaqşa`,
    'kbd': `Адыгэбзэ (Къэбэрдейбзэ) Adıgăbză (Qăbărdeĭbză)`,
    'kha': `কা কতিয়েন খাশি`,
    'khm': `ភាសាខ្មែរ Phiəsaakhmær`,
    'kik': `Gĩkũyũ`,
    'kin': `Ikinyarwanda`,
    'kir': `кыргызча kırgızça / кыргыз тили kırgız tili`,
    'kok': `कोंकणी`,
    'kom': `Коми кыв`,
    'kor': `한국어 Han'gug'ô`,
    'kpe': `Kpɛlɛwoo`,
    'krc': `Къарачай-Малкъар тил / Таулу тил`,
    'krl': `karjal / kariela / karjala`,
    'kru': `कुड़ुख़`,
    'kum': `къумукъ тил / qumuq til`,
    'kur': `kurdî / کوردی`,
    'lad': `Judeo-español`,
    'lah': `بھارت کا`,
    'lao': `ພາສາລາວ Phasalaw`,
    'lat': `Lingua latīna`,
    'lav': `Latviešu valoda`,
    'lez': `Лезги чӏал`,
    'lim': `Lèmburgs`,
    'lit': `lietuvių kalba`,
    'lol': `Lomongo`,
    'ltz': `Lëtzebuergesch`,
    'lua': `Tshiluba`,
    'lub': `Kiluba`,
    'lug': `Luganda`,
    'lui': `Cham'teela`,
    'lun': `Chilunda`,
    'luo': `Dholuo`,
    'lus': `Mizo ṭawng`,
    'mad': `Madhura`,
    'mag': `मगही`,
    'mah': `Kajin M̧ajeļ`,
    'mai': `मैथिली; মৈথিলী`,
    'mak': `Basa Mangkasara' / ᨅᨔ ᨆᨀᨔᨑ`,
    'mal': `മലയാളം Malayāļã`,
    'man': `Mandi'nka kango`,
    'mar': `मराठी Marāţhī`,
    'mas': `ɔl`,
    'mdf': `мокшень кяль`,
    'men': `Mɛnde yia`,
    'mga': `Gaoidhealg`,
    'mic': `Míkmawísimk`,
    'min': `Baso Minang`,
    'mkd': `македонски јазик makedonski jazik`,
    'mlt': `Malti`,
    'mnc': `ᠮᠠᠨᠵᡠ ᡤᡳᠰᡠᠨ Manju gisun`,
    'moh': `Kanien’kéha`,
    'mon': `монгол хэл mongol xel / ᠮᠣᠩᠭᠣᠯ ᠬᠡᠯᠡ`,
    'mos': `Mooré`,
    'mri': `Te Reo Māori`,
    'msa': `Bahasa Melayu`,
    'mus': `Mvskoke`,
    'mwl': `mirandés / lhéngua mirandesa`,
    'mwr': `मारवाड़ी`,
    'mya': `မြန်မာစာ Mrãmācā / မြန်မာစကား Mrãmākā:`,
    'nap': `napulitano`,
    'nau': `dorerin Naoero`,
    'nav': `Diné bizaad / Naabeehó bizaad`,
    'nbl': `isiNdebele seSewula`,
    'nde': `siNdebele saseNyakatho`,
    'ndo': `ndonga`,
    'nds': `Plattdütsch / Plattdüütsch`,
    'nep': `नेपाली भाषा Nepālī bhāśā`,
    'new': `नेपाल भाषा / नेवाः भाय्`,
    'nia': `Li Niha`,
    'niu': `ko e vagahau Niuē`,
    'nld': `Nederlands; Vlaams`,
    'nno': `norsk nynorsk`,
    'nob': `norsk bokmål`,
    'nog': `Ногай тили`,
    'nor': `norsk`,
    'nso': `Sesotho sa Leboa`,
    'nub': `لغات نوبية`,
    'nya': `Chichewa; Chinyanja`,
    'nyo': `Runyoro`,
    'oci': `occitan; lenga d'òc`,
    'ori': `ଓଡ଼ିଆ`,
    'orm': `Afaan Oromoo`,
    'osa': `Wazhazhe ie / 𐓏𐓘𐓻𐓘𐓻𐓟 𐒻𐓟`,
    'oss': `Ирон ӕвзаг Iron ævzag`,
    'pag': `Salitan Pangasinan`,
    'pam': `Amánung Kapampangan / Amánung Sísuan`,
    'pan': `ਪੰਜਾਬੀ / پنجابی Pãjābī`,
    'pap': `Papiamentu`,
    'pau': `a tekoi er a Belau`,
    'pol': `Język polski`,
    'por': `português`,
    'pus': `پښتو Pax̌tow`,
    'que': `Runa simi / kichwa simi / Nuna shimi`,
    'raj': `राजस्थानी`,
    'rap': `Vananga rapa nui`,
    'rar': `Māori Kūki 'Āirani`,
    'roh': `Rumantsch / Rumàntsch / Romauntsch / Romontsch`,
    'rom': `romani čhib`,
    'ron': `limba română`,
    'run': `Ikirundi`,
    'rup': `armãneashce / armãneashti / rrãmãneshti`,
    'rus': `русский язык russkiĭ âzık`,
    'sad': `Sandaweeki`,
    'sag': `yângâ tî sängö`,
    'sah': `Сахалыы`,
    'sam': `ארמית`,
    'san': `संस्कृतम् Sąskŕtam / 𑌸𑌂𑌸𑍍𑌕𑍃𑌤𑌮𑍍`,
    'sat': `ᱥᱟᱱᱛᱟᱲᱤ`,
    'scn': `Sicilianu`,
    'sco': `Braid Scots; Lallans`,
    'shn': `ၵႂၢမ်းတႆးယႂ်`,
    'sid': `Sidaamu Afoo`,
    'sin': `සිංහල Sĩhala`,
    'slk': `slovenčina / slovenský jazyk`,
    'slv': `slovenski jezik / slovenščina`,
    'sma': `Åarjelsaemien gïele`,
    'sme': `davvisámegiella`,
    'smj': `julevsámegiella`,
    'smn': `anarâškielâ`,
    'smo': `Gagana faʻa Sāmoa`,
    'sms': `sääʹmǩiõll`,
    'sna': `chiShona`,
    'snd': `سنڌي / सिन्धी / ਸਿੰਧੀ`,
    'snk': `Sooninkanxanne`,
    'som': `af Soomaali`,
    'sot': `Sesotho [southern]`,
    'spa': `español / castellano`,
    'sqi': `Shqip`,
    'srd': `sardu / limba sarda / lingua sarda`,
    'srp': `српски / srpski`,
    'srr': `Seereer`,
    'ssw': `siSwati`,
    'suk': `Kɪsukuma`,
    'sun': `ᮘᮞ ᮞᮥᮔ᮪ᮓ / Basa Sunda`,
    'sus': `Sosoxui`,
    'swa': `Kiswahili`,
    'swe': `svenska`,
    'syr': `ܠܫܢܐ ܣܘܪܝܝܐ Lešānā Suryāyā`,
    'tah': `Reo Tahiti / Reo Mā'ohi`,
    'tam': `தமிழ் Tamił`,
    'tat': `татар теле / tatar tele / تاتار`,
    'tel': `తెలుగు Telugu`,
    'tem': `KʌThemnɛ`,
    'ter': `Terêna`,
    'tet': `Lia-Tetun`,
    'tgk': `тоҷикӣ toçikī`,
    'tgl': `Wikang Tagalog`,
    'tha': `ภาษาไทย Phasathay`,
    'tig': `ትግረ / ትግሬ / ኻሳ / ትግራይት`,
    'tir': `ትግርኛ`,
    'tli': `Lingít`,
    'tog': `chiTonga`,
    'ton': `lea faka-Tonga`,
    'tsn': `Setswana`,
    'tso': `Xitsonga`,
    'tuk': `Türkmençe / Түркменче / تورکمن تیلی تورکمنچ; türkmen dili / түркмен дили`,
    'tum': `chiTumbuka`,
    'tur': `Türkçe`,
    'tvl': `Te Ggana Tuuvalu / Te Gagana Tuuvalu`,
    'tyv': `тыва дыл`,
    'udm': `ئۇيغۇرچە / ئۇيغۇر تىلى`,
    'uig': `Українська мова / Українська`,
    'ukr': `Úmbúndú`,
    'urd': `اُردُو Urduw`,
    'uzb': `Oʻzbekcha / Ózbekça / ўзбекча / ئوزبېچه; oʻzbek tili / ўзбек тили / ئوبېک تیلی`,
    'vai': `ꕙꔤ`,
    'ven': `Tshivenḓa`,
    'vie': `Tiếng Việt`,
    'vot': `vađđa ceeli`,
    'war': `Winaray / Samareño / Lineyte-Samarnon / Binisayâ nga Winaray / Binisayâ nga Samar-Leyte / “Binisayâ nga Waray”`,
    'was': `wá:šiw ʔítlu`,
    'wln': `Walon`,
    'xal': `Хальмг келн / Xaľmg keln`,
    'xho': `isiXhosa`,
    'yid': `אידיש /  יידיש / ייִדיש / Yidiš`,
    'yor': `èdè Yorùbá`,
    'zap': `Diidxazá/Dizhsa`,
    'zen': `Tuḍḍungiyya`,
    'zgh': `ⵜⴰⵎⴰⵣⵉⵖⵜ ⵜⴰⵏⴰⵡⴰⵢⵜ`,
    'zha': `Vahcuengh / 話僮`,
    'zho': `中文 Zhōngwén / 汉语 / 漢語 Hànyǔ`,
    'zul': `isiZulu`,
    'zun': `Shiwi'ma`,
    'zza': `kirmanckî / dimilkî / kirdkî / zazakî`,
}

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

    const userLanguages = useMemo(() => session.languages ?? ['eng'], [session]);
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
    const onOpen = useCallback((event: MouseEvent<HTMLButtonElement>) => setAnchorEl(event.currentTarget), []);
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
                    overflow: 'auto',
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
                        value={searchString}
                        onChange={updateSearchString}
                    />
                    <List>
                        {languageOptions.map((option: [string, string], index: number) => (
                            <ListItem button onClick={() => { handleSelect(option[0]); onClose(); }} key={index}>
                                {option[1]}
                            </ListItem>
                        ))}
                    </List>
                </Stack>
            </Popover>
            {/* Selected language label */}
            <Tooltip title="Select language" placement="top">
                <Stack direction="row" spacing={0} sx={{
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
                    <Typography variant="body2" sx={{ paddingRight: '8px' }}>
                        {language.toLocaleUpperCase()}
                    </Typography>
                    {/* Drop down or drop up icon */}
                    <IconButton size="large" aria-label="language-select" sx={{ padding: '4px' }} onClick={onOpen}>
                        {open ? <ArrowDropUpIcon sx={{ fill: 'white' }} /> : <ArrowDropDownIcon sx={{ fill: 'white' }} />}
                    </IconButton>
                </Stack>
            </Tooltip>
        </>
    )
}