export const feed_home = {
  "fieldName": "home",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "home",
        "loc": {
          "start": 2736,
          "end": 2740
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 2741,
              "end": 2746
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 2749,
                "end": 2754
              }
            },
            "loc": {
              "start": 2748,
              "end": 2754
            }
          },
          "loc": {
            "start": 2741,
            "end": 2754
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "notes",
              "loc": {
                "start": 2762,
                "end": 2767
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Note_list",
                    "loc": {
                      "start": 2781,
                      "end": 2790
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2778,
                    "end": 2790
                  }
                }
              ],
              "loc": {
                "start": 2768,
                "end": 2796
              }
            },
            "loc": {
              "start": 2762,
              "end": 2796
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminders",
              "loc": {
                "start": 2801,
                "end": 2810
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Reminder_full",
                    "loc": {
                      "start": 2824,
                      "end": 2837
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2821,
                    "end": 2837
                  }
                }
              ],
              "loc": {
                "start": 2811,
                "end": 2843
              }
            },
            "loc": {
              "start": 2801,
              "end": 2843
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "resources",
              "loc": {
                "start": 2848,
                "end": 2857
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Resource_list",
                    "loc": {
                      "start": 2871,
                      "end": 2884
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2868,
                    "end": 2884
                  }
                }
              ],
              "loc": {
                "start": 2858,
                "end": 2890
              }
            },
            "loc": {
              "start": 2848,
              "end": 2890
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "schedules",
              "loc": {
                "start": 2895,
                "end": 2904
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Schedule_list",
                    "loc": {
                      "start": 2918,
                      "end": 2931
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2915,
                    "end": 2931
                  }
                }
              ],
              "loc": {
                "start": 2905,
                "end": 2937
              }
            },
            "loc": {
              "start": 2895,
              "end": 2937
            }
          }
        ],
        "loc": {
          "start": 2756,
          "end": 2941
        }
      },
      "loc": {
        "start": 2736,
        "end": 2941
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 9,
          "end": 19
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 23,
            "end": 28
          }
        },
        "loc": {
          "start": 23,
          "end": 28
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 31,
                "end": 33
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 31,
              "end": 33
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 34,
                "end": 44
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 34,
              "end": 44
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 45,
                "end": 55
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 45,
              "end": 55
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 56,
                "end": 61
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 56,
              "end": 61
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 62,
                "end": 67
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 62,
              "end": 67
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 68,
                "end": 73
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 87,
                        "end": 99
                      }
                    },
                    "loc": {
                      "start": 87,
                      "end": 99
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 113,
                            "end": 129
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 110,
                          "end": 129
                        }
                      }
                    ],
                    "loc": {
                      "start": 100,
                      "end": 135
                    }
                  },
                  "loc": {
                    "start": 80,
                    "end": 135
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 147,
                        "end": 151
                      }
                    },
                    "loc": {
                      "start": 147,
                      "end": 151
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 165,
                            "end": 173
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 162,
                          "end": 173
                        }
                      }
                    ],
                    "loc": {
                      "start": 152,
                      "end": 179
                    }
                  },
                  "loc": {
                    "start": 140,
                    "end": 179
                  }
                }
              ],
              "loc": {
                "start": 74,
                "end": 181
              }
            },
            "loc": {
              "start": 68,
              "end": 181
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 182,
                "end": 185
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 192,
                      "end": 201
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 192,
                    "end": 201
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 206,
                      "end": 215
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 206,
                    "end": 215
                  }
                }
              ],
              "loc": {
                "start": 186,
                "end": 217
              }
            },
            "loc": {
              "start": 182,
              "end": 217
            }
          }
        ],
        "loc": {
          "start": 29,
          "end": 219
        }
      },
      "loc": {
        "start": 0,
        "end": 219
      }
    },
    "Note_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_list",
        "loc": {
          "start": 229,
          "end": 238
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 242,
            "end": 246
          }
        },
        "loc": {
          "start": 242,
          "end": 246
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 249,
                "end": 257
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 264,
                      "end": 276
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 287,
                            "end": 289
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 287,
                          "end": 289
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 298,
                            "end": 306
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 298,
                          "end": 306
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 315,
                            "end": 326
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 315,
                          "end": 326
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 335,
                            "end": 339
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 335,
                          "end": 339
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "text",
                          "loc": {
                            "start": 348,
                            "end": 352
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 348,
                          "end": 352
                        }
                      }
                    ],
                    "loc": {
                      "start": 277,
                      "end": 358
                    }
                  },
                  "loc": {
                    "start": 264,
                    "end": 358
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 363,
                      "end": 365
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 363,
                    "end": 365
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 370,
                      "end": 380
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 370,
                    "end": 380
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 385,
                      "end": 395
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 385,
                    "end": 395
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 400,
                      "end": 408
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 400,
                    "end": 408
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 413,
                      "end": 422
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 413,
                    "end": 422
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 427,
                      "end": 439
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 427,
                    "end": 439
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 444,
                      "end": 456
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 444,
                    "end": 456
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 461,
                      "end": 473
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 461,
                    "end": 473
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 478,
                      "end": 481
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canComment",
                          "loc": {
                            "start": 492,
                            "end": 502
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 492,
                          "end": 502
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 511,
                            "end": 518
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 511,
                          "end": 518
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 527,
                            "end": 536
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 527,
                          "end": 536
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 545,
                            "end": 554
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 545,
                          "end": 554
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 563,
                            "end": 572
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 563,
                          "end": 572
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 581,
                            "end": 587
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 581,
                          "end": 587
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 596,
                            "end": 603
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 596,
                          "end": 603
                        }
                      }
                    ],
                    "loc": {
                      "start": 482,
                      "end": 609
                    }
                  },
                  "loc": {
                    "start": 478,
                    "end": 609
                  }
                }
              ],
              "loc": {
                "start": 258,
                "end": 611
              }
            },
            "loc": {
              "start": 249,
              "end": 611
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 612,
                "end": 614
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 612,
              "end": 614
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 615,
                "end": 625
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 615,
              "end": 625
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 626,
                "end": 636
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 626,
              "end": 636
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 637,
                "end": 646
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 637,
              "end": 646
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 647,
                "end": 658
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 647,
              "end": 658
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 659,
                "end": 665
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 675,
                      "end": 685
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 672,
                    "end": 685
                  }
                }
              ],
              "loc": {
                "start": 666,
                "end": 687
              }
            },
            "loc": {
              "start": 659,
              "end": 687
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 688,
                "end": 693
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 707,
                        "end": 719
                      }
                    },
                    "loc": {
                      "start": 707,
                      "end": 719
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 733,
                            "end": 749
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 730,
                          "end": 749
                        }
                      }
                    ],
                    "loc": {
                      "start": 720,
                      "end": 755
                    }
                  },
                  "loc": {
                    "start": 700,
                    "end": 755
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 767,
                        "end": 771
                      }
                    },
                    "loc": {
                      "start": 767,
                      "end": 771
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 785,
                            "end": 793
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 782,
                          "end": 793
                        }
                      }
                    ],
                    "loc": {
                      "start": 772,
                      "end": 799
                    }
                  },
                  "loc": {
                    "start": 760,
                    "end": 799
                  }
                }
              ],
              "loc": {
                "start": 694,
                "end": 801
              }
            },
            "loc": {
              "start": 688,
              "end": 801
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 802,
                "end": 813
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 802,
              "end": 813
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 814,
                "end": 828
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 814,
              "end": 828
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 829,
                "end": 834
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 829,
              "end": 834
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 835,
                "end": 844
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 835,
              "end": 844
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 845,
                "end": 849
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 859,
                      "end": 867
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 856,
                    "end": 867
                  }
                }
              ],
              "loc": {
                "start": 850,
                "end": 869
              }
            },
            "loc": {
              "start": 845,
              "end": 869
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 870,
                "end": 884
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 870,
              "end": 884
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 885,
                "end": 890
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 885,
              "end": 890
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 891,
                "end": 894
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 901,
                      "end": 910
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 901,
                    "end": 910
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 915,
                      "end": 926
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 915,
                    "end": 926
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 931,
                      "end": 942
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 931,
                    "end": 942
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 947,
                      "end": 956
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 947,
                    "end": 956
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 961,
                      "end": 968
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 961,
                    "end": 968
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 973,
                      "end": 981
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 973,
                    "end": 981
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 986,
                      "end": 998
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 986,
                    "end": 998
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1003,
                      "end": 1011
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1003,
                    "end": 1011
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1016,
                      "end": 1024
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1016,
                    "end": 1024
                  }
                }
              ],
              "loc": {
                "start": 895,
                "end": 1026
              }
            },
            "loc": {
              "start": 891,
              "end": 1026
            }
          }
        ],
        "loc": {
          "start": 247,
          "end": 1028
        }
      },
      "loc": {
        "start": 220,
        "end": 1028
      }
    },
    "Organization_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_nav",
        "loc": {
          "start": 1038,
          "end": 1054
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 1058,
            "end": 1070
          }
        },
        "loc": {
          "start": 1058,
          "end": 1070
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1073,
                "end": 1075
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1073,
              "end": 1075
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 1076,
                "end": 1082
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1076,
              "end": 1082
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1083,
                "end": 1086
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 1093,
                      "end": 1106
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1093,
                    "end": 1106
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1111,
                      "end": 1120
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1111,
                    "end": 1120
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1125,
                      "end": 1136
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1125,
                    "end": 1136
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 1141,
                      "end": 1150
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1141,
                    "end": 1150
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1155,
                      "end": 1164
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1155,
                    "end": 1164
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1169,
                      "end": 1176
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1169,
                    "end": 1176
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1181,
                      "end": 1193
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1181,
                    "end": 1193
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1198,
                      "end": 1206
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1198,
                    "end": 1206
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 1211,
                      "end": 1225
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1236,
                            "end": 1238
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1236,
                          "end": 1238
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 1247,
                            "end": 1257
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1247,
                          "end": 1257
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 1266,
                            "end": 1276
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1266,
                          "end": 1276
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 1285,
                            "end": 1292
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1285,
                          "end": 1292
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 1301,
                            "end": 1312
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1301,
                          "end": 1312
                        }
                      }
                    ],
                    "loc": {
                      "start": 1226,
                      "end": 1318
                    }
                  },
                  "loc": {
                    "start": 1211,
                    "end": 1318
                  }
                }
              ],
              "loc": {
                "start": 1087,
                "end": 1320
              }
            },
            "loc": {
              "start": 1083,
              "end": 1320
            }
          }
        ],
        "loc": {
          "start": 1071,
          "end": 1322
        }
      },
      "loc": {
        "start": 1029,
        "end": 1322
      }
    },
    "Reminder_full": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Reminder_full",
        "loc": {
          "start": 1332,
          "end": 1345
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Reminder",
          "loc": {
            "start": 1349,
            "end": 1357
          }
        },
        "loc": {
          "start": 1349,
          "end": 1357
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1360,
                "end": 1362
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1360,
              "end": 1362
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1363,
                "end": 1373
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1363,
              "end": 1373
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1374,
                "end": 1384
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1374,
              "end": 1384
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1385,
                "end": 1389
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1385,
              "end": 1389
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1390,
                "end": 1401
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1390,
              "end": 1401
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dueDate",
              "loc": {
                "start": 1402,
                "end": 1409
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1402,
              "end": 1409
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 1410,
                "end": 1415
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1410,
              "end": 1415
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1416,
                "end": 1426
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1416,
              "end": 1426
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderItems",
              "loc": {
                "start": 1427,
                "end": 1440
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1447,
                      "end": 1449
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1447,
                    "end": 1449
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1454,
                      "end": 1464
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1454,
                    "end": 1464
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1469,
                      "end": 1479
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1469,
                    "end": 1479
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1484,
                      "end": 1488
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1484,
                    "end": 1488
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1493,
                      "end": 1504
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1493,
                    "end": 1504
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dueDate",
                    "loc": {
                      "start": 1509,
                      "end": 1516
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1509,
                    "end": 1516
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "index",
                    "loc": {
                      "start": 1521,
                      "end": 1526
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1521,
                    "end": 1526
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 1531,
                      "end": 1541
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1531,
                    "end": 1541
                  }
                }
              ],
              "loc": {
                "start": 1441,
                "end": 1543
              }
            },
            "loc": {
              "start": 1427,
              "end": 1543
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderList",
              "loc": {
                "start": 1544,
                "end": 1556
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1563,
                      "end": 1565
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1563,
                    "end": 1565
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1570,
                      "end": 1580
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1570,
                    "end": 1580
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1585,
                      "end": 1595
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1585,
                    "end": 1595
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusMode",
                    "loc": {
                      "start": 1600,
                      "end": 1609
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 1620,
                            "end": 1626
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 1641,
                                  "end": 1643
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1641,
                                "end": 1643
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 1656,
                                  "end": 1661
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1656,
                                "end": 1661
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 1674,
                                  "end": 1679
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1674,
                                "end": 1679
                              }
                            }
                          ],
                          "loc": {
                            "start": 1627,
                            "end": 1689
                          }
                        },
                        "loc": {
                          "start": 1620,
                          "end": 1689
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 1698,
                            "end": 1706
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Schedule_common",
                                "loc": {
                                  "start": 1724,
                                  "end": 1739
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 1721,
                                "end": 1739
                              }
                            }
                          ],
                          "loc": {
                            "start": 1707,
                            "end": 1749
                          }
                        },
                        "loc": {
                          "start": 1698,
                          "end": 1749
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1758,
                            "end": 1760
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1758,
                          "end": 1760
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1769,
                            "end": 1773
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1769,
                          "end": 1773
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1782,
                            "end": 1793
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1782,
                          "end": 1793
                        }
                      }
                    ],
                    "loc": {
                      "start": 1610,
                      "end": 1799
                    }
                  },
                  "loc": {
                    "start": 1600,
                    "end": 1799
                  }
                }
              ],
              "loc": {
                "start": 1557,
                "end": 1801
              }
            },
            "loc": {
              "start": 1544,
              "end": 1801
            }
          }
        ],
        "loc": {
          "start": 1358,
          "end": 1803
        }
      },
      "loc": {
        "start": 1323,
        "end": 1803
      }
    },
    "Resource_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Resource_list",
        "loc": {
          "start": 1813,
          "end": 1826
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Resource",
          "loc": {
            "start": 1830,
            "end": 1838
          }
        },
        "loc": {
          "start": 1830,
          "end": 1838
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1841,
                "end": 1843
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1841,
              "end": 1843
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 1844,
                "end": 1849
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1844,
              "end": 1849
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "link",
              "loc": {
                "start": 1850,
                "end": 1854
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1850,
              "end": 1854
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "usedFor",
              "loc": {
                "start": 1855,
                "end": 1862
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1855,
              "end": 1862
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1863,
                "end": 1875
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1882,
                      "end": 1884
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1882,
                    "end": 1884
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1889,
                      "end": 1897
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1889,
                    "end": 1897
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1902,
                      "end": 1913
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1902,
                    "end": 1913
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1918,
                      "end": 1922
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1918,
                    "end": 1922
                  }
                }
              ],
              "loc": {
                "start": 1876,
                "end": 1924
              }
            },
            "loc": {
              "start": 1863,
              "end": 1924
            }
          }
        ],
        "loc": {
          "start": 1839,
          "end": 1926
        }
      },
      "loc": {
        "start": 1804,
        "end": 1926
      }
    },
    "Schedule_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_common",
        "loc": {
          "start": 1936,
          "end": 1951
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 1955,
            "end": 1963
          }
        },
        "loc": {
          "start": 1955,
          "end": 1963
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1966,
                "end": 1968
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1966,
              "end": 1968
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1969,
                "end": 1979
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1969,
              "end": 1979
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1980,
                "end": 1990
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1980,
              "end": 1990
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 1991,
                "end": 2000
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1991,
              "end": 2000
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 2001,
                "end": 2008
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2001,
              "end": 2008
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 2009,
                "end": 2017
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2009,
              "end": 2017
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 2018,
                "end": 2028
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2035,
                      "end": 2037
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2035,
                    "end": 2037
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 2042,
                      "end": 2059
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2042,
                    "end": 2059
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 2064,
                      "end": 2076
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2064,
                    "end": 2076
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 2081,
                      "end": 2091
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2081,
                    "end": 2091
                  }
                }
              ],
              "loc": {
                "start": 2029,
                "end": 2093
              }
            },
            "loc": {
              "start": 2018,
              "end": 2093
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 2094,
                "end": 2105
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2112,
                      "end": 2114
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2112,
                    "end": 2114
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 2119,
                      "end": 2133
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2119,
                    "end": 2133
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 2138,
                      "end": 2146
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2138,
                    "end": 2146
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 2151,
                      "end": 2160
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2151,
                    "end": 2160
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 2165,
                      "end": 2175
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2165,
                    "end": 2175
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 2180,
                      "end": 2185
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2180,
                    "end": 2185
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 2190,
                      "end": 2197
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2190,
                    "end": 2197
                  }
                }
              ],
              "loc": {
                "start": 2106,
                "end": 2199
              }
            },
            "loc": {
              "start": 2094,
              "end": 2199
            }
          }
        ],
        "loc": {
          "start": 1964,
          "end": 2201
        }
      },
      "loc": {
        "start": 1927,
        "end": 2201
      }
    },
    "Schedule_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_list",
        "loc": {
          "start": 2211,
          "end": 2224
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 2228,
            "end": 2236
          }
        },
        "loc": {
          "start": 2228,
          "end": 2236
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 2239,
                "end": 2245
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 2255,
                      "end": 2265
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2252,
                    "end": 2265
                  }
                }
              ],
              "loc": {
                "start": 2246,
                "end": 2267
              }
            },
            "loc": {
              "start": 2239,
              "end": 2267
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2268,
                "end": 2270
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2268,
              "end": 2270
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2271,
                "end": 2281
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2271,
              "end": 2281
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2282,
                "end": 2292
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2282,
              "end": 2292
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 2293,
                "end": 2302
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2293,
              "end": 2302
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 2303,
                "end": 2310
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2303,
              "end": 2310
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 2311,
                "end": 2319
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2311,
              "end": 2319
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 2320,
                "end": 2330
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2337,
                      "end": 2339
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2337,
                    "end": 2339
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 2344,
                      "end": 2361
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2344,
                    "end": 2361
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 2366,
                      "end": 2378
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2366,
                    "end": 2378
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 2383,
                      "end": 2393
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2383,
                    "end": 2393
                  }
                }
              ],
              "loc": {
                "start": 2331,
                "end": 2395
              }
            },
            "loc": {
              "start": 2320,
              "end": 2395
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 2396,
                "end": 2407
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2414,
                      "end": 2416
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2414,
                    "end": 2416
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 2421,
                      "end": 2435
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2421,
                    "end": 2435
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 2440,
                      "end": 2448
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2440,
                    "end": 2448
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 2453,
                      "end": 2462
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2453,
                    "end": 2462
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 2467,
                      "end": 2477
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2467,
                    "end": 2477
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 2482,
                      "end": 2487
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2482,
                    "end": 2487
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 2492,
                      "end": 2499
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2492,
                    "end": 2499
                  }
                }
              ],
              "loc": {
                "start": 2408,
                "end": 2501
              }
            },
            "loc": {
              "start": 2396,
              "end": 2501
            }
          }
        ],
        "loc": {
          "start": 2237,
          "end": 2503
        }
      },
      "loc": {
        "start": 2202,
        "end": 2503
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 2513,
          "end": 2521
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 2525,
            "end": 2528
          }
        },
        "loc": {
          "start": 2525,
          "end": 2528
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2531,
                "end": 2533
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2531,
              "end": 2533
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2534,
                "end": 2544
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2534,
              "end": 2544
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 2545,
                "end": 2548
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2545,
              "end": 2548
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2549,
                "end": 2558
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2549,
              "end": 2558
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2559,
                "end": 2571
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2578,
                      "end": 2580
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2578,
                    "end": 2580
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2585,
                      "end": 2593
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2585,
                    "end": 2593
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2598,
                      "end": 2609
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2598,
                    "end": 2609
                  }
                }
              ],
              "loc": {
                "start": 2572,
                "end": 2611
              }
            },
            "loc": {
              "start": 2559,
              "end": 2611
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2612,
                "end": 2615
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isOwn",
                    "loc": {
                      "start": 2622,
                      "end": 2627
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2622,
                    "end": 2627
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2632,
                      "end": 2644
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2632,
                    "end": 2644
                  }
                }
              ],
              "loc": {
                "start": 2616,
                "end": 2646
              }
            },
            "loc": {
              "start": 2612,
              "end": 2646
            }
          }
        ],
        "loc": {
          "start": 2529,
          "end": 2648
        }
      },
      "loc": {
        "start": 2504,
        "end": 2648
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 2658,
          "end": 2666
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 2670,
            "end": 2674
          }
        },
        "loc": {
          "start": 2670,
          "end": 2674
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2677,
                "end": 2679
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2677,
              "end": 2679
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 2680,
                "end": 2685
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2680,
              "end": 2685
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2686,
                "end": 2690
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2686,
              "end": 2690
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2691,
                "end": 2697
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2691,
              "end": 2697
            }
          }
        ],
        "loc": {
          "start": 2675,
          "end": 2699
        }
      },
      "loc": {
        "start": 2649,
        "end": 2699
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "home",
      "loc": {
        "start": 2707,
        "end": 2711
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 2713,
              "end": 2718
            }
          },
          "loc": {
            "start": 2712,
            "end": 2718
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "HomeInput",
              "loc": {
                "start": 2720,
                "end": 2729
              }
            },
            "loc": {
              "start": 2720,
              "end": 2729
            }
          },
          "loc": {
            "start": 2720,
            "end": 2730
          }
        },
        "directives": [],
        "loc": {
          "start": 2712,
          "end": 2730
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "home",
            "loc": {
              "start": 2736,
              "end": 2740
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 2741,
                  "end": 2746
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 2749,
                    "end": 2754
                  }
                },
                "loc": {
                  "start": 2748,
                  "end": 2754
                }
              },
              "loc": {
                "start": 2741,
                "end": 2754
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "notes",
                  "loc": {
                    "start": 2762,
                    "end": 2767
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Note_list",
                        "loc": {
                          "start": 2781,
                          "end": 2790
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 2778,
                        "end": 2790
                      }
                    }
                  ],
                  "loc": {
                    "start": 2768,
                    "end": 2796
                  }
                },
                "loc": {
                  "start": 2762,
                  "end": 2796
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "reminders",
                  "loc": {
                    "start": 2801,
                    "end": 2810
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Reminder_full",
                        "loc": {
                          "start": 2824,
                          "end": 2837
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 2821,
                        "end": 2837
                      }
                    }
                  ],
                  "loc": {
                    "start": 2811,
                    "end": 2843
                  }
                },
                "loc": {
                  "start": 2801,
                  "end": 2843
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "resources",
                  "loc": {
                    "start": 2848,
                    "end": 2857
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Resource_list",
                        "loc": {
                          "start": 2871,
                          "end": 2884
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 2868,
                        "end": 2884
                      }
                    }
                  ],
                  "loc": {
                    "start": 2858,
                    "end": 2890
                  }
                },
                "loc": {
                  "start": 2848,
                  "end": 2890
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "schedules",
                  "loc": {
                    "start": 2895,
                    "end": 2904
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Schedule_list",
                        "loc": {
                          "start": 2918,
                          "end": 2931
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 2915,
                        "end": 2931
                      }
                    }
                  ],
                  "loc": {
                    "start": 2905,
                    "end": 2937
                  }
                },
                "loc": {
                  "start": 2895,
                  "end": 2937
                }
              }
            ],
            "loc": {
              "start": 2756,
              "end": 2941
            }
          },
          "loc": {
            "start": 2736,
            "end": 2941
          }
        }
      ],
      "loc": {
        "start": 2732,
        "end": 2943
      }
    },
    "loc": {
      "start": 2701,
      "end": 2943
    }
  },
  "variableValues": {},
  "path": {
    "key": "feed_home"
  }
};
