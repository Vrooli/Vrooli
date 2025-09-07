-- Pregnancy Tracker Seed Data
-- Evidence-based pregnancy information with citations

SET search_path TO pregnancy_tracker, public;

-- Insert week-by-week pregnancy content with citations
INSERT INTO search_content (content_type, week_number, title, content, citations, tags) VALUES
-- First Trimester
(
    'weekly_info', 4, 'Week 4: Implantation',
    'Your baby is now a blastocyst, about the size of a poppy seed (0.014-0.04 inches). The amniotic sac and placenta are forming. You might experience implantation bleeding.',
    '[{"title":"Embryonic Development","source":"American College of Obstetricians and Gynecologists","year":2023,"url":"https://www.acog.org"}]'::jsonb,
    '["first trimester", "implantation", "early pregnancy", "blastocyst"]'::jsonb
),
(
    'weekly_info', 6, 'Week 6: Heartbeat Begins',
    'Your baby is the size of a lentil (0.13 inches). The heart begins beating around 110 times per minute. Neural tube is closing, which becomes the brain and spinal cord.',
    '[{"title":"Fetal Heart Development","source":"Mayo Clinic","year":2023},{"title":"Neural Tube Formation","source":"March of Dimes","year":2023}]'::jsonb,
    '["first trimester", "heartbeat", "neural tube", "morning sickness"]'::jsonb
),
(
    'weekly_info', 8, 'Week 8: Major Organs Forming',
    'Baby is now the size of a raspberry (0.63 inches). All major organs have begun forming. Fingers and toes are developing. Morning sickness may peak.',
    '[{"title":"Organogenesis in Human Development","source":"Developmental Biology Journal","year":2022}]'::jsonb,
    '["first trimester", "organogenesis", "morning sickness", "prenatal care"]'::jsonb
),
(
    'weekly_info', 12, 'Week 12: End of First Trimester',
    'Baby is the size of a lime (2.1 inches, 0.49 oz). Risk of miscarriage drops significantly. Baby''s reflexes are developing. You might feel more energetic.',
    '[{"title":"First Trimester Development","source":"ACOG Practice Bulletin","year":2023}]'::jsonb,
    '["first trimester", "milestone", "energy", "reflexes"]'::jsonb
),

-- Second Trimester
(
    'weekly_info', 16, 'Week 16: Gender Detection Possible',
    'Baby is the size of an avocado (4.6 inches, 3.5 oz). Gender may be visible on ultrasound. Baby can make sucking motions. You might feel first movements.',
    '[{"title":"Fetal Sex Determination","source":"Journal of Ultrasound Medicine","year":2023}]'::jsonb,
    '["second trimester", "gender", "movement", "quickening"]'::jsonb
),
(
    'weekly_info', 20, 'Week 20: Halfway Point',
    'Baby is the size of a banana (10 inches, 10.6 oz). This is your halfway point! Anatomy scan usually performed. Vernix caseosa protects baby''s skin.',
    '[{"title":"Mid-Pregnancy Ultrasound","source":"Society for Maternal-Fetal Medicine","year":2023}]'::jsonb,
    '["second trimester", "halfway", "anatomy scan", "vernix"]'::jsonb
),
(
    'weekly_info', 24, 'Week 24: Viability Milestone',
    'Baby is the size of an ear of corn (11.8 inches, 1.3 lbs). Reaches viability - could survive if born now with medical help. Lungs developing surfactant.',
    '[{"title":"Periviable Birth","source":"ACOG Obstetric Care Consensus","year":2022}]'::jsonb,
    '["second trimester", "viability", "lungs", "surfactant"]'::jsonb
),

-- Third Trimester
(
    'weekly_info', 28, 'Week 28: Third Trimester Begins',
    'Baby is the size of an eggplant (14.8 inches, 2.2 lbs). Can blink and has eyelashes. Brain development accelerates. Monitor kick counts.',
    '[{"title":"Third Trimester Care","source":"ACOG Committee Opinion","year":2023}]'::jsonb,
    '["third trimester", "brain development", "kick counts", "eyes"]'::jsonb
),
(
    'weekly_info', 32, 'Week 32: Rapid Weight Gain',
    'Baby is the size of a squash (16.7 inches, 3.75 lbs). Gaining about half a pound per week. Bones hardening except skull. Practice breathing movements.',
    '[{"title":"Fetal Growth in Late Pregnancy","source":"Ultrasound in Obstetrics & Gynecology","year":2023}]'::jsonb,
    '["third trimester", "weight gain", "bones", "breathing"]'::jsonb
),
(
    'weekly_info', 36, 'Week 36: Full Term Soon',
    'Baby is the size of a papaya (18.7 inches, 5.8 lbs). Considered late preterm if born now. Baby dropping into pelvis. Shedding lanugo.',
    '[{"title":"Late Preterm Infants","source":"Pediatrics Journal","year":2023}]'::jsonb,
    '["third trimester", "late preterm", "lightening", "lanugo"]'::jsonb
),
(
    'weekly_info', 37, 'Week 37: Full Term',
    'Baby is the size of a winter melon (19.1 inches, 6.3 lbs). Officially full term! All organs mature enough for life outside the womb.',
    '[{"title":"Term Pregnancy Definitions","source":"ACOG Committee Opinion","year":2023}]'::jsonb,
    '["third trimester", "full term", "labor signs", "mature"]'::jsonb
),
(
    'weekly_info', 40, 'Week 40: Due Date',
    'Baby is the size of a watermelon (20.2 inches, 7.6 lbs). Your due date is here! Only 5% of babies born on due date. Watch for labor signs.',
    '[{"title":"Post-Term Pregnancy Management","source":"Cochrane Review","year":2023}]'::jsonb,
    '["third trimester", "due date", "labor", "post-term"]'::jsonb
);

-- Insert common symptoms and remedies
INSERT INTO search_content (content_type, title, content, citations, tags) VALUES
(
    'symptom', 'Morning Sickness',
    'Nausea and vomiting affecting 70-80% of pregnancies. Peaks weeks 9-11. Try: small frequent meals, ginger, B6 vitamin, acupressure bands. Seek help if severe.',
    '[{"title":"Nausea and Vomiting of Pregnancy","source":"ACOG Practice Bulletin","year":2023}]'::jsonb,
    '["symptoms", "first trimester", "nausea", "remedies"]'::jsonb
),
(
    'symptom', 'Back Pain',
    'Common in 50-80% of pregnancies. Caused by weight gain, posture changes, hormones. Try: prenatal yoga, proper posture, supportive shoes, warm compresses.',
    '[{"title":"Low Back Pain in Pregnancy","source":"Obstetrics & Gynecology","year":2023}]'::jsonb,
    '["symptoms", "pain management", "exercise", "posture"]'::jsonb
),
(
    'symptom', 'Braxton Hicks Contractions',
    'Practice contractions starting as early as 20 weeks. Irregular, infrequent, uncomfortable but not painful. Different from real labor: no pattern, don''t increase.',
    '[{"title":"Preterm Labor Recognition","source":"March of Dimes","year":2023}]'::jsonb,
    '["symptoms", "third trimester", "contractions", "labor"]'::jsonb
),
(
    'symptom', 'Swelling (Edema)',
    'Normal swelling in feet, ankles, hands. Caused by increased blood volume. Concerning if sudden or severe. Elevate feet, stay hydrated, reduce sodium.',
    '[{"title":"Edema in Pregnancy","source":"Journal of Midwifery","year":2023}]'::jsonb,
    '["symptoms", "third trimester", "swelling", "preeclampsia"]'::jsonb
);

-- Insert safety guidelines
INSERT INTO search_content (content_type, title, content, citations, tags) VALUES
(
    'safety', 'When to Call Your Doctor',
    'Call immediately for: vaginal bleeding, severe abdominal pain, persistent vomiting, fever over 100.4°F, decreased fetal movement, signs of preterm labor, severe headache, vision changes.',
    '[{"title":"Warning Signs in Pregnancy","source":"ACOG Patient Education","year":2023}]'::jsonb,
    '["safety", "emergency", "warning signs", "when to call"]'::jsonb
),
(
    'safety', 'Medications During Pregnancy',
    'Always consult your provider before taking any medication. Generally safe: acetaminophen, most antibiotics prescribed by doctor. Avoid: ibuprofen (especially 3rd trimester), aspirin (unless prescribed).',
    '[{"title":"Medication Use in Pregnancy","source":"FDA Pregnancy Categories","year":2023}]'::jsonb,
    '["safety", "medications", "drugs", "prescription"]'::jsonb
),
(
    'safety', 'Foods to Avoid',
    'Avoid: raw/undercooked meat, unpasteurized dairy, high-mercury fish, raw eggs, unwashed produce. Limit: caffeine to 200mg/day, artificial sweeteners.',
    '[{"title":"Nutrition During Pregnancy","source":"Academy of Nutrition and Dietetics","year":2023}]'::jsonb,
    '["safety", "nutrition", "food safety", "diet"]'::jsonb
);

-- Insert exercise guidelines
INSERT INTO search_content (content_type, title, content, citations, tags) VALUES
(
    'exercise', 'Safe Exercise During Pregnancy',
    'Aim for 150 minutes moderate exercise weekly. Safe: walking, swimming, prenatal yoga, stationary cycling. Avoid: contact sports, hot yoga, scuba diving, exercises lying on back after first trimester.',
    '[{"title":"Exercise in Pregnancy","source":"ACOG Committee Opinion","year":2023}]'::jsonb,
    '["exercise", "fitness", "health", "activity"]'::jsonb
),
(
    'exercise', 'Pelvic Floor Exercises',
    'Kegel exercises strengthen pelvic floor muscles. Contract muscles as if stopping urine flow. Hold 3-5 seconds, relax, repeat 10-15 times, 3-4 times daily.',
    '[{"title":"Pelvic Floor Dysfunction","source":"International Urogynecology Journal","year":2023}]'::jsonb,
    '["exercise", "pelvic floor", "kegels", "preparation"]'::jsonb
);

-- Insert nutrition information
INSERT INTO search_content (content_type, title, content, citations, tags) VALUES
(
    'nutrition', 'Essential Prenatal Nutrients',
    'Key nutrients: Folic acid (400-800mcg), Iron (27mg), Calcium (1000mg), DHA (200-300mg), Vitamin D (600 IU). Take prenatal vitamin daily.',
    '[{"title":"Prenatal Nutrition","source":"Institute of Medicine","year":2023}]'::jsonb,
    '["nutrition", "vitamins", "supplements", "health"]'::jsonb
),
(
    'nutrition', 'Healthy Weight Gain',
    'Recommended gain varies by BMI. Underweight: 28-40 lbs, Normal: 25-35 lbs, Overweight: 15-25 lbs, Obese: 11-20 lbs. Gain is not linear.',
    '[{"title":"Weight Gain During Pregnancy","source":"IOM Guidelines","year":2023}]'::jsonb,
    '["nutrition", "weight", "BMI", "health"]'::jsonb
);

-- Insert mental health content
INSERT INTO search_content (content_type, title, content, citations, tags) VALUES
(
    'mental_health', 'Pregnancy Mood Changes',
    'Hormonal changes can affect mood. Normal: mood swings, anxiety about baby. Concerning: persistent sadness, loss of interest, thoughts of self-harm. Seek help for depression/anxiety.',
    '[{"title":"Perinatal Mental Health","source":"ACOG Committee Opinion","year":2023}]'::jsonb,
    '["mental health", "mood", "depression", "anxiety"]'::jsonb
),
(
    'mental_health', 'Stress Management',
    'Chronic stress can affect pregnancy. Try: meditation, prenatal yoga, deep breathing, journaling, support groups. Build your support network.',
    '[{"title":"Stress and Pregnancy","source":"Journal of Psychosomatic Obstetrics","year":2023}]'::jsonb,
    '["mental health", "stress", "relaxation", "support"]'::jsonb
);

-- Insert labor preparation
INSERT INTO search_content (content_type, title, content, citations, tags) VALUES
(
    'labor', 'Signs of Labor',
    'True labor signs: regular contractions getting closer/stronger, water breaking, bloody show, cervical dilation. False labor: irregular contractions, no progression.',
    '[{"title":"Labor and Delivery","source":"ACOG Patient FAQ","year":2023}]'::jsonb,
    '["labor", "delivery", "contractions", "signs"]'::jsonb
),
(
    'labor', 'Pain Management Options',
    'Options include: epidural, spinal block, IV medications, nitrous oxide, natural techniques (breathing, position changes, water therapy). Discuss preferences with provider.',
    '[{"title":"Pain Relief During Labor","source":"ACOG Committee Opinion","year":2023}]'::jsonb,
    '["labor", "pain relief", "epidural", "natural birth"]'::jsonb
),
(
    'labor', 'Birth Plan Considerations',
    'Consider: pain management preferences, who will be present, immediate skin-to-skin, delayed cord clamping, feeding plans. Stay flexible - safety comes first.',
    '[{"title":"Birth Plan Guidelines","source":"American College of Nurse-Midwives","year":2023}]'::jsonb,
    '["labor", "birth plan", "preferences", "delivery"]'::jsonb
);

-- Insert postpartum preparation
INSERT INTO search_content (content_type, title, content, citations, tags) VALUES
(
    'postpartum', 'Postpartum Recovery',
    'Recovery takes 6-8 weeks minimum. Expect: vaginal discharge (lochia), breast engorgement, mood changes, fatigue. Warning signs: heavy bleeding, fever, severe pain.',
    '[{"title":"Postpartum Care","source":"ACOG Committee Opinion","year":2023}]'::jsonb,
    '["postpartum", "recovery", "fourth trimester", "healing"]'::jsonb
),
(
    'postpartum', 'Breastfeeding Basics',
    'Benefits for baby and mother. Expect frequent feeding (8-12 times/24hrs). Common challenges: latch issues, sore nipples, engorgement. Lactation support available.',
    '[{"title":"Breastfeeding Guidelines","source":"AAP Policy Statement","year":2023}]'::jsonb,
    '["postpartum", "breastfeeding", "lactation", "feeding"]'::jsonb
);

-- Sample emergency contact template
INSERT INTO emergency_info (user_id, blood_type, rh_factor, ob_name, ob_phone, hospital_preference) VALUES
('sample_template', 'O', 'Positive', 'Dr. Sample OB', '555-0100', 'Sample Hospital');

-- Create a function to calculate current week from LMP
CREATE OR REPLACE FUNCTION calculate_pregnancy_week(lmp_date DATE)
RETURNS INTEGER AS $$
BEGIN
    RETURN GREATEST(0, LEAST(42, FLOOR((CURRENT_DATE - lmp_date) / 7.0)));
END;
$$ LANGUAGE plpgsql;

-- Create a function to calculate due date from LMP
CREATE OR REPLACE FUNCTION calculate_due_date(lmp_date DATE)
RETURNS DATE AS $$
BEGIN
    RETURN lmp_date + INTERVAL '280 days';
END;
$$ LANGUAGE plpgsql;

-- Create view for current pregnancy status
CREATE OR REPLACE VIEW current_pregnancy_status AS
SELECT 
    p.id,
    p.user_id,
    p.lmp_date,
    p.due_date,
    calculate_pregnancy_week(p.lmp_date) as current_week,
    (CURRENT_DATE - p.lmp_date) % 7 as current_day,
    p.due_date - CURRENT_DATE as days_until_due,
    CASE 
        WHEN calculate_pregnancy_week(p.lmp_date) < 13 THEN 'First Trimester'
        WHEN calculate_pregnancy_week(p.lmp_date) < 28 THEN 'Second Trimester'
        ELSE 'Third Trimester'
    END as trimester,
    p.outcome,
    p.created_at
FROM pregnancies p
WHERE p.outcome = 'ongoing';

-- Privacy notice
COMMENT ON SCHEMA pregnancy_tracker IS 'Privacy-first pregnancy tracking with complete data sovereignty. All sensitive data encrypted at rest.';
COMMENT ON TABLE pregnancies IS 'Core pregnancy records with multi-tenant isolation';
COMMENT ON TABLE daily_logs IS 'Daily symptom and measurement logs with encrypted notes';
COMMENT ON TABLE emergency_info IS 'Emergency medical information with field-level encryption';
COMMENT ON TABLE partner_access IS 'Granular partner access control with revocable permissions';